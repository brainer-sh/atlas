// Package search implements symbol search across indexed repositories.
package search

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/brainer.sh/atlas/internal/storage"
)

const defaultLimit = 20

// ResultItem is a single symbol match returned by Search.
type ResultItem struct {
	Name      string  `json:"name"`
	Kind      string  `json:"kind"`
	Signature string  `json:"signature"`
	Doc       string  `json:"doc"`
	File      string  `json:"file"`
	LineStart int64   `json:"line_start"`
	LineEnd   int64   `json:"line_end"`
	Score     float64 `json:"score"`
}

// Result is the full response for a search query.
type Result struct {
	Query   string       `json:"query"`
	Results []ResultItem `json:"results"`
}

// Search runs FTS across all *.db files in atlasDir and returns ranked results.
// Scores are normalized to [0,1] where 1 is the best match.
func Search(atlasDir, query string) (*Result, error) {
	entries, err := os.ReadDir(atlasDir)
	if os.IsNotExist(err) {
		return &Result{Query: query, Results: []ResultItem{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("search: read dir %s: %w", atlasDir, err)
	}

	var raw []storage.SearchResult
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".db" {
			continue
		}
		batch, err := searchDB(filepath.Join(atlasDir, e.Name()), query)
		if err != nil {
			return nil, err
		}
		raw = append(raw, batch...)
	}

	items := toItems(raw)
	return &Result{Query: query, Results: items}, nil
}

func searchDB(dbPath, query string) ([]storage.SearchResult, error) {
	store, err := storage.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("search: open %s: %w", dbPath, err)
	}
	defer store.Close()
	results, err := store.Search(query, defaultLimit)
	if err != nil {
		return nil, fmt.Errorf("search: query %s: %w", dbPath, err)
	}
	return results, nil
}

// toItems converts raw storage results to ResultItems with normalized scores.
// bm25() returns negative values; more negative means more relevant.
// We map the most relevant result to score 1.0 and least relevant to 0.0.
func toItems(raw []storage.SearchResult) []ResultItem {
	if len(raw) == 0 {
		return []ResultItem{}
	}

	// Find score range.
	minScore, maxScore := raw[0].Score, raw[0].Score
	for _, r := range raw[1:] {
		if r.Score < minScore {
			minScore = r.Score
		}
		if r.Score > maxScore {
			maxScore = r.Score
		}
	}
	span := maxScore - minScore

	items := make([]ResultItem, len(raw))
	for i, r := range raw {
		var normalized float64
		if span == 0 {
			normalized = 1.0
		} else {
			// minScore is most relevant; invert so it maps to 1.0.
			normalized = (maxScore - r.Score) / span
		}
		items[i] = ResultItem{
			Name:      r.Name,
			Kind:      r.Kind,
			Signature: r.Signature,
			Doc:       r.Doc,
			File:      r.FilePath,
			LineStart: r.LineStart,
			LineEnd:   r.LineEnd,
			Score:     math.Round(normalized*100) / 100,
		}
	}
	return items
}
