package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/brainer.sh/atlas/internal/storage"
)

// ExploreResult holds detail about a single symbol.
type ExploreResult struct {
	Symbol    string   `json:"symbol"`
	Kind      string   `json:"kind"`
	File      string   `json:"file"`
	LineStart int64    `json:"line_start"`
	LineEnd   int64    `json:"line_end"`
	Signature string   `json:"signature"`
	Doc       string   `json:"doc"`
	Callers   []string `json:"callers"`
	Callees   []string `json:"callees"`
}

// ExploreSymbol searches all *.db files in atlasDir for the first exact match
// of symbolName and returns its detail. Returns nil if not found.
func ExploreSymbol(atlasDir, symbolName string) (*ExploreResult, error) {
	entries, err := os.ReadDir(atlasDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tools/explore: read dir %s: %w", atlasDir, err)
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".db" {
			continue
		}
		detail, err := symbolFromDB(filepath.Join(atlasDir, e.Name()), symbolName)
		if err != nil {
			return nil, err
		}
		if detail != nil {
			return &ExploreResult{
				Symbol:    detail.Name,
				Kind:      detail.Kind,
				File:      detail.FilePath,
				LineStart: detail.LineStart,
				LineEnd:   detail.LineEnd,
				Signature: detail.Signature,
				Doc:       detail.Doc,
				Callers:   []string{},
				Callees:   []string{},
			}, nil
		}
	}
	return nil, nil
}

func symbolFromDB(dbPath, symbolName string) (*storage.SymbolDetail, error) {
	store, err := storage.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("tools/explore: open %s: %w", dbPath, err)
	}
	defer store.Close()
	detail, err := store.GetSymbolByName(symbolName)
	if err != nil {
		return nil, fmt.Errorf("tools/explore: query %s: %w", dbPath, err)
	}
	return detail, nil
}
