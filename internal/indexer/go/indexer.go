// Package goindexer extracts symbols from Go source files using tree-sitter.
package goindexer

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

//go:embed queries/symbols.scm
var symbolsQuery string

//go:embed queries/call_sites.scm
var callSitesQueryStr string

// Symbol represents an extracted code symbol.
type Symbol struct {
	Name      string
	Kind      string // function | method | struct | interface
	Signature string
	Doc       string
	LineStart uint
	LineEnd   uint
}

// CallSite represents a function or method call found in source.
type CallSite struct {
	CalleeName string
	Line       uint
}

// FileIndex holds all extracted information from a single Go source file.
type FileIndex struct {
	Package   string
	Imports   []string
	Symbols   []Symbol
	CallSites []CallSite
}

// Indexer parses Go source files and extracts symbols using tree-sitter.
type Indexer struct {
	parser         *sitter.Parser
	query          *sitter.Query
	callSitesQuery *sitter.Query
}

// New creates a new Go Indexer.
func New() (*Indexer, error) {
	lang := sitter.NewLanguage(tree_sitter_go.Language())

	parser := sitter.NewParser()
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("indexer/go: set language: %w", err)
	}

	query, qErr := sitter.NewQuery(lang, symbolsQuery)
	if qErr != nil {
		return nil, fmt.Errorf("indexer/go: compile query: %w", qErr)
	}

	csQuery, csErr := sitter.NewQuery(lang, callSitesQueryStr)
	if csErr != nil {
		query.Close()
		return nil, fmt.Errorf("indexer/go: compile call sites query: %w", csErr)
	}

	return &Indexer{parser: parser, query: query, callSitesQuery: csQuery}, nil
}

// Close releases resources held by the Indexer.
func (idx *Indexer) Close() {
	idx.callSitesQuery.Close()
	idx.query.Close()
	idx.parser.Close()
}

// IndexFile reads and indexes a single Go source file.
func (idx *Indexer) IndexFile(path string) (*FileIndex, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("indexer/go: read %s: %w", path, err)
	}
	return idx.IndexSource(src)
}

// IndexSource indexes Go source from a byte slice.
func (idx *Indexer) IndexSource(src []byte) (*FileIndex, error) {
	tree := idx.parser.Parse(src, nil)
	defer tree.Close()
	root := tree.RootNode()

	return &FileIndex{
		Package:   extractPackage(root, src),
		Imports:   extractImports(root, src),
		Symbols:   idx.extractSymbols(root, src),
		CallSites: idx.extractCallSites(root, src),
	}, nil
}

func extractPackage(root *sitter.Node, src []byte) string {
	for i := range root.NamedChildCount() {
		child := root.NamedChild(i)
		if child.Kind() == "package_clause" && child.NamedChildCount() > 0 {
			return child.NamedChild(0).Utf8Text(src)
		}
	}
	return ""
}

func extractImports(root *sitter.Node, src []byte) []string {
	var imports []string
	for i := range root.NamedChildCount() {
		child := root.NamedChild(i)
		if child.Kind() != "import_declaration" {
			continue
		}
		for j := range child.NamedChildCount() {
			spec := child.NamedChild(j)
			if spec.Kind() != "import_spec" {
				continue
			}
			if path := spec.ChildByFieldName("path"); path != nil {
				imports = append(imports, strings.Trim(path.Utf8Text(src), `"`))
			}
		}
	}
	return imports
}

func (idx *Indexer) extractSymbols(root *sitter.Node, src []byte) []Symbol {
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(idx.query, root, src)
	captureNames := idx.query.CaptureNames()

	var symbols []Symbol
	for {
		match := matches.Next()
		if match == nil {
			break
		}

		var kind string
		var defNode, nameNode *sitter.Node

		for i := range match.Captures {
			cap := match.Captures[i]
			capName := captureNames[cap.Index]
			node := cap.Node
			switch capName {
			case "definition.function":
				kind = "function"
				defNode = &node
			case "definition.method":
				kind = "method"
				defNode = &node
			case "definition.struct":
				kind = "struct"
				defNode = &node
			case "definition.interface":
				kind = "interface"
				defNode = &node
			case "name":
				nameNode = &node
			}
		}

		if defNode == nil || nameNode == nil {
			continue
		}

		symbols = append(symbols, Symbol{
			Name:      nameNode.Utf8Text(src),
			Kind:      kind,
			Signature: buildSignature(defNode, src),
			Doc:       extractDoc(defNode, src),
			LineStart: defNode.StartPosition().Row + 1,
			LineEnd:   defNode.EndPosition().Row + 1,
		})
	}

	return symbols
}

func (idx *Indexer) extractCallSites(root *sitter.Node, src []byte) []CallSite {
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(idx.callSitesQuery, root, src)
	captureNames := idx.callSitesQuery.CaptureNames()

	var sites []CallSite
	for {
		match := matches.Next()
		if match == nil {
			break
		}
		for i := range match.Captures {
			cap := match.Captures[i]
			if captureNames[cap.Index] != "callee.name" {
				continue
			}
			node := cap.Node
			sites = append(sites, CallSite{
				CalleeName: node.Utf8Text(src),
				Line:       node.StartPosition().Row + 1,
			})
		}
	}
	return sites
}

func buildSignature(node *sitter.Node, src []byte) string {
	body := node.ChildByFieldName("body")
	if body == nil {
		return strings.SplitN(node.Utf8Text(src), "\n", 2)[0]
	}
	return strings.TrimSpace(string(src[node.StartByte():body.StartByte()]))
}

func extractDoc(node *sitter.Node, src []byte) string {
	prev := node.PrevNamedSibling()
	if prev == nil || prev.Kind() != "comment" {
		return ""
	}
	text := prev.Utf8Text(src)
	text = strings.TrimPrefix(text, "// ")
	text = strings.TrimPrefix(text, "/*")
	text = strings.TrimSuffix(text, "*/")
	return strings.TrimSpace(text)
}
