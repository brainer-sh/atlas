// Package cindexer extracts symbols from C source files using tree-sitter.
package cindexer

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_c "github.com/tree-sitter/tree-sitter-c/bindings/go"
)

//go:embed queries/symbols.scm
var symbolsQuery string

// Symbol represents an extracted code symbol.
type Symbol struct {
	Name      string
	Kind      string // function | struct | enum | macro
	Signature string
	Doc       string
	LineStart uint
	LineEnd   uint
}

// FileIndex holds all extracted information from a single C source file.
type FileIndex struct {
	IsHeader bool
	Symbols  []Symbol
}

// Indexer parses C source files and extracts symbols using tree-sitter.
type Indexer struct {
	parser *sitter.Parser
	query  *sitter.Query
}

// New creates a new C Indexer.
func New() (*Indexer, error) {
	lang := sitter.NewLanguage(tree_sitter_c.Language())

	parser := sitter.NewParser()
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("indexer/c: set language: %w", err)
	}

	query, qErr := sitter.NewQuery(lang, symbolsQuery)
	if qErr != nil {
		return nil, fmt.Errorf("indexer/c: compile query: %w", qErr)
	}

	return &Indexer{parser: parser, query: query}, nil
}

// Close releases resources held by the Indexer.
func (idx *Indexer) Close() {
	idx.query.Close()
	idx.parser.Close()
}

// IndexFile reads and indexes a single C source file.
func (idx *Indexer) IndexFile(path string) (*FileIndex, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("indexer/c: read %s: %w", path, err)
	}
	fi, err := idx.IndexSource(src)
	if err != nil {
		return nil, err
	}
	fi.IsHeader = strings.HasSuffix(path, ".h")
	return fi, nil
}

// IndexSource indexes C source from a byte slice.
func (idx *Indexer) IndexSource(src []byte) (*FileIndex, error) {
	tree := idx.parser.Parse(src, nil)
	defer tree.Close()
	root := tree.RootNode()

	return &FileIndex{
		Symbols: idx.extractSymbols(root, src),
	}, nil
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
			case "definition.struct":
				kind = "struct"
				defNode = &node
			case "definition.enum":
				kind = "enum"
				defNode = &node
			case "definition.macro":
				kind = "macro"
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
