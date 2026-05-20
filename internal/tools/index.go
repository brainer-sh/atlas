// Package tools implements the business logic for Atlas MCP tools.
package tools

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	cindexer "github.com/brainer.sh/atlas/internal/indexer/c"
	cppindexer "github.com/brainer.sh/atlas/internal/indexer/cpp"
	goindexer "github.com/brainer.sh/atlas/internal/indexer/go"
	"github.com/brainer.sh/atlas/internal/storage"
)

// IndexResult holds the outcome of an index or reindex operation.
type IndexResult struct {
	Repo           string
	Path           string
	FilesIndexed   int
	SymbolsIndexed int
	DurationMs     int64
}

// rawCallSite is a call site as extracted by the indexer, before symbol ID resolution.
type rawCallSite struct {
	calleeName string
	line       int64
}

// parseFunc converts a file path + source bytes to storage symbols and raw call sites.
type parseFunc func(path string, src []byte) ([]storage.Symbol, []rawCallSite, error)

// IndexRepo fully indexes a repository and stores results in store.
// The language is auto-detected from the file extensions present.
func IndexRepo(repoPath string, store *storage.Store) (*IndexResult, error) {
	start := time.Now()
	name := filepath.Base(repoPath)
	lang := detectLang(repoPath)

	repoID, err := store.UpsertRepo(storage.Repo{
		Path:      repoPath,
		Name:      name,
		Lang:      lang,
		IndexedAt: start.Unix(),
	})
	if err != nil {
		return nil, fmt.Errorf("tools/index: upsert repo: %w", err)
	}

	parse, walk, cleanup, err := langPipeline(lang)
	if err != nil {
		return nil, fmt.Errorf("tools/index: init %s pipeline: %w", lang, err)
	}
	defer cleanup()

	var filesIndexed, symbolsIndexed int
	err = walk(repoPath, func(filePath string, content []byte, mtime int64) error {
		n, err := indexFile(store, parse, repoID, repoPath, filePath, content, mtime)
		if err != nil {
			return err
		}
		filesIndexed++
		symbolsIndexed += n
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("tools/index: walk %s: %w", repoPath, err)
	}

	return &IndexResult{
		Repo:           name,
		Path:           repoPath,
		FilesIndexed:   filesIndexed,
		SymbolsIndexed: symbolsIndexed,
		DurationMs:     time.Since(start).Milliseconds(),
	}, nil
}

// ReindexRepo re-indexes only files that changed since the last run.
// Falls back to a full index if the repo has never been indexed.
func ReindexRepo(repoPath string, store *storage.Store) (*IndexResult, error) {
	start := time.Now()
	name := filepath.Base(repoPath)

	repo, err := store.GetRepoByPath(repoPath)
	if err != nil {
		return nil, fmt.Errorf("tools/reindex: get repo: %w", err)
	}
	if repo == nil {
		return IndexRepo(repoPath, store)
	}

	repoID := repo.ID
	lang := repo.Lang
	if _, err := store.UpsertRepo(storage.Repo{
		Path:      repoPath,
		Name:      name,
		Lang:      lang,
		IndexedAt: start.Unix(),
	}); err != nil {
		return nil, fmt.Errorf("tools/reindex: upsert repo: %w", err)
	}

	parse, walk, cleanup, err := langPipeline(lang)
	if err != nil {
		return nil, fmt.Errorf("tools/reindex: init %s pipeline: %w", lang, err)
	}
	defer cleanup()

	var filesIndexed, symbolsIndexed int
	err = walk(repoPath, func(filePath string, content []byte, mtime int64) error {
		relPath, _ := filepath.Rel(repoPath, filePath)
		hash := hashBytes(content)

		existing, err := store.GetFile(repoID, relPath)
		if err != nil {
			return fmt.Errorf("get file %s: %w", relPath, err)
		}
		if existing != nil && existing.Hash == hash && existing.Mtime == mtime {
			return nil // unchanged
		}

		n, err := indexFile(store, parse, repoID, repoPath, filePath, content, mtime)
		if err != nil {
			return err
		}
		filesIndexed++
		symbolsIndexed += n
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("tools/reindex: walk %s: %w", repoPath, err)
	}

	return &IndexResult{
		Repo:           name,
		Path:           repoPath,
		FilesIndexed:   filesIndexed,
		SymbolsIndexed: symbolsIndexed,
		DurationMs:     time.Since(start).Milliseconds(),
	}, nil
}

// langPipeline returns the parse function, walker, and cleanup for a language.
func langPipeline(lang string) (parseFunc, walkFunc, func(), error) {
	switch lang {
	case "c":
		idx, err := cindexer.New()
		if err != nil {
			return nil, nil, nil, err
		}
		parse := func(_ string, src []byte) ([]storage.Symbol, []rawCallSite, error) {
			fi, err := idx.IndexSource(src)
			if err != nil {
				return nil, nil, err
			}
			return cSymbols(fi.Symbols), cRawSites(fi.CallSites), nil
		}
		return parse, walkCFiles, idx.Close, nil

	case "cpp":
		idx, err := cppindexer.New()
		if err != nil {
			return nil, nil, nil, err
		}
		parse := func(_ string, src []byte) ([]storage.Symbol, []rawCallSite, error) {
			fi, err := idx.IndexSource(src)
			if err != nil {
				return nil, nil, err
			}
			return cppSymbols(fi.Symbols), cppRawSites(fi.CallSites), nil
		}
		return parse, walkCppFiles, idx.Close, nil

	default: // "go"
		idx, err := goindexer.New()
		if err != nil {
			return nil, nil, nil, err
		}
		parse := func(_ string, src []byte) ([]storage.Symbol, []rawCallSite, error) {
			fi, err := idx.IndexSource(src)
			if err != nil {
				return nil, nil, err
			}
			return goSymbols(fi.Symbols), goRawSites(fi.CallSites), nil
		}
		return parse, walkGoFiles, idx.Close, nil
	}
}

// indexFile parses a single file and stores its symbols and call sites. Returns symbol count.
func indexFile(store *storage.Store, parse parseFunc, repoID int64, repoPath, filePath string, content []byte, mtime int64) (int, error) {
	relPath, _ := filepath.Rel(repoPath, filePath)

	fileID, err := store.UpsertFile(storage.File{
		RepoID: repoID,
		Path:   relPath,
		Hash:   hashBytes(content),
		Mtime:  mtime,
	})
	if err != nil {
		return 0, fmt.Errorf("upsert file %s: %w", relPath, err)
	}

	// Delete call sites before symbols (call_sites.caller_symbol_id references symbols).
	if err := store.DeleteCallSitesForFile(fileID); err != nil {
		return 0, fmt.Errorf("delete call sites for %s: %w", relPath, err)
	}
	if err := store.DeleteSymbolsForFile(fileID); err != nil {
		return 0, fmt.Errorf("delete symbols for %s: %w", relPath, err)
	}

	syms, rawSites, err := parse(filePath, content)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", relPath, err)
	}
	for i := range syms {
		syms[i].FileID = fileID
		syms[i].RepoID = repoID
	}

	if err := store.InsertSymbols(syms); err != nil {
		return 0, fmt.Errorf("insert symbols for %s: %w", relPath, err)
	}

	if len(rawSites) > 0 {
		symsDB, err := store.GetSymbolsByFileID(fileID)
		if err != nil {
			return 0, fmt.Errorf("get symbols for %s: %w", relPath, err)
		}
		sites := resolveCallSites(rawSites, symsDB, fileID)
		if err := store.InsertCallSites(sites); err != nil {
			return 0, fmt.Errorf("insert call sites for %s: %w", relPath, err)
		}
	}

	return len(syms), nil
}

// resolveCallSites matches raw call sites (by line) to the enclosing symbol in syms.
func resolveCallSites(raw []rawCallSite, syms []storage.Symbol, fileID int64) []storage.CallSite {
	var sites []storage.CallSite
	for _, r := range raw {
		for _, sym := range syms {
			if r.line >= sym.LineStart && r.line <= sym.LineEnd {
				sites = append(sites, storage.CallSite{
					CallerSymbolID: sym.ID,
					CalleeName:     r.calleeName,
					FileID:         fileID,
					Line:           r.line,
				})
				break
			}
		}
	}
	return sites
}

// walkFunc walks a repo and calls fn for each source file.
type walkFunc func(root string, fn func(path string, content []byte, mtime int64) error) error

func walkGoFiles(root string, fn func(path string, content []byte, mtime int64) error) error {
	return walkFiles(root, []string{".go"}, func(name string) bool {
		return strings.HasSuffix(name, "_test.go")
	}, fn)
}

func walkCFiles(root string, fn func(path string, content []byte, mtime int64) error) error {
	return walkFiles(root, []string{".c", ".h"}, nil, fn)
}

func walkCppFiles(root string, fn func(path string, content []byte, mtime int64) error) error {
	return walkFiles(root, []string{".cpp", ".cxx", ".cc", ".hpp", ".hxx", ".hh", ".h", ".c"}, nil, fn)
}

func walkFiles(root string, exts []string, skip func(name string) bool, fn func(string, []byte, int64) error) error {
	extSet := make(map[string]bool, len(exts))
	for _, e := range exts {
		extSet[e] = true
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		if !extSet[filepath.Ext(path)] {
			return nil
		}
		if skip != nil && skip(d.Name()) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return fn(path, content, info.ModTime().Unix())
	})
}

// detectLang returns the dominant language of a repo by counting source files.
func detectLang(repoPath string) string {
	counts := map[string]int{}
	_ = filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".go":
			counts["go"]++
		case ".cpp", ".cxx", ".cc", ".hpp", ".hxx", ".hh":
			counts["cpp"]++
		case ".c", ".h":
			counts["c"]++
		}
		return nil
	})
	best := "go"
	bestN := 0
	for lang, n := range counts {
		if n > bestN {
			best, bestN = lang, n
		}
	}
	return best
}

// Symbol converters.

func goSymbols(in []goindexer.Symbol) []storage.Symbol {
	out := make([]storage.Symbol, len(in))
	for i, s := range in {
		out[i] = storage.Symbol{
			Name:      s.Name,
			Kind:      s.Kind,
			Signature: s.Signature,
			Doc:       s.Doc,
			LineStart: int64(s.LineStart),
			LineEnd:   int64(s.LineEnd),
		}
	}
	return out
}

func cSymbols(in []cindexer.Symbol) []storage.Symbol {
	out := make([]storage.Symbol, len(in))
	for i, s := range in {
		out[i] = storage.Symbol{
			Name:      s.Name,
			Kind:      s.Kind,
			Signature: s.Signature,
			Doc:       s.Doc,
			LineStart: int64(s.LineStart),
			LineEnd:   int64(s.LineEnd),
		}
	}
	return out
}

func cppSymbols(in []cppindexer.Symbol) []storage.Symbol {
	out := make([]storage.Symbol, len(in))
	for i, s := range in {
		out[i] = storage.Symbol{
			Name:      s.Name,
			Kind:      s.Kind,
			Signature: s.Signature,
			Doc:       s.Doc,
			LineStart: int64(s.LineStart),
			LineEnd:   int64(s.LineEnd),
		}
	}
	return out
}

func goRawSites(in []goindexer.CallSite) []rawCallSite {
	out := make([]rawCallSite, len(in))
	for i, cs := range in {
		out[i] = rawCallSite{calleeName: cs.CalleeName, line: int64(cs.Line)}
	}
	return out
}

func cRawSites(in []cindexer.CallSite) []rawCallSite {
	out := make([]rawCallSite, len(in))
	for i, cs := range in {
		out[i] = rawCallSite{calleeName: cs.CalleeName, line: int64(cs.Line)}
	}
	return out
}

func cppRawSites(in []cppindexer.CallSite) []rawCallSite {
	out := make([]rawCallSite, len(in))
	for i, cs := range in {
		out[i] = rawCallSite{calleeName: cs.CalleeName, line: int64(cs.Line)}
	}
	return out
}

func hashBytes(b []byte) string {
	h := sha1.Sum(b)
	return hex.EncodeToString(h[:])
}
