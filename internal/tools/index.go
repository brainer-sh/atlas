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

// IndexRepo fully indexes all Go files in repoPath and stores results in store.
func IndexRepo(repoPath string, store *storage.Store) (*IndexResult, error) {
	start := time.Now()
	name := filepath.Base(repoPath)

	repoID, err := store.UpsertRepo(storage.Repo{
		Path:      repoPath,
		Name:      name,
		Lang:      "go",
		IndexedAt: start.Unix(),
	})
	if err != nil {
		return nil, fmt.Errorf("tools/index: upsert repo: %w", err)
	}

	idx, err := goindexer.New()
	if err != nil {
		return nil, fmt.Errorf("tools/index: create indexer: %w", err)
	}
	defer idx.Close()

	var filesIndexed, symbolsIndexed int
	err = walkGoFiles(repoPath, func(filePath string, content []byte, mtime int64) error {
		n, err := indexFile(store, idx, repoID, repoPath, filePath, content, mtime)
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

// ReindexRepo re-indexes only Go files that changed since the last run.
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

	// Update indexed_at but keep the existing ID.
	repoID := repo.ID
	if _, err := store.UpsertRepo(storage.Repo{
		Path:      repoPath,
		Name:      name,
		Lang:      repo.Lang,
		IndexedAt: start.Unix(),
	}); err != nil {
		return nil, fmt.Errorf("tools/reindex: upsert repo: %w", err)
	}

	idx, err := goindexer.New()
	if err != nil {
		return nil, fmt.Errorf("tools/reindex: create indexer: %w", err)
	}
	defer idx.Close()

	var filesIndexed, symbolsIndexed int
	err = walkGoFiles(repoPath, func(filePath string, content []byte, mtime int64) error {
		relPath, _ := filepath.Rel(repoPath, filePath)
		hash := hashBytes(content)

		existing, err := store.GetFile(repoID, relPath)
		if err != nil {
			return fmt.Errorf("get file %s: %w", relPath, err)
		}
		if existing != nil && existing.Hash == hash && existing.Mtime == mtime {
			return nil // unchanged
		}

		n, err := indexFile(store, idx, repoID, repoPath, filePath, content, mtime)
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

// indexFile parses a single file and stores its symbols. Returns symbol count.
func indexFile(store *storage.Store, idx *goindexer.Indexer, repoID int64, repoPath, filePath string, content []byte, mtime int64) (int, error) {
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

	if err := store.DeleteSymbolsForFile(fileID); err != nil {
		return 0, fmt.Errorf("delete symbols for %s: %w", relPath, err)
	}

	fi, err := idx.IndexSource(content)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", relPath, err)
	}

	syms := make([]storage.Symbol, len(fi.Symbols))
	for i, s := range fi.Symbols {
		syms[i] = storage.Symbol{
			FileID:    fileID,
			RepoID:    repoID,
			Name:      s.Name,
			Kind:      s.Kind,
			Signature: s.Signature,
			Doc:       s.Doc,
			LineStart: int64(s.LineStart),
			LineEnd:   int64(s.LineEnd),
		}
	}

	if err := store.InsertSymbols(syms); err != nil {
		return 0, fmt.Errorf("insert symbols for %s: %w", relPath, err)
	}
	return len(syms), nil
}

func walkGoFiles(root string, fn func(path string, content []byte, mtime int64) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
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

func hashBytes(b []byte) string {
	h := sha1.Sum(b)
	return hex.EncodeToString(h[:])
}
