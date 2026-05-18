package search

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"sort"

	"github.com/brainer.sh/atlas/internal/embeddings"
	"github.com/brainer.sh/atlas/internal/storage"
)

const (
	hybridFTSWeight    = 0.6
	hybridVectorWeight = 0.4
)

// HybridSearch combines FTS and vector similarity across all *.db files in atlasDir.
// If embedder is nil, it falls back to FTS-only search.
func HybridSearch(ctx context.Context, atlasDir, query string, embedder embeddings.Embedder, limit int) (*Result, error) {
	dbs, err := listDBs(atlasDir)
	if err != nil {
		return &Result{Query: query, Results: []ResultItem{}}, nil
	}

	var queryVec []float32
	if embedder != nil {
		vecs, err := embedder.Embed(ctx, []string{query})
		if err != nil {
			return nil, fmt.Errorf("search: embed query: %w", err)
		}
		queryVec = vecs[0]
	}

	var allItems []ResultItem
	for _, dbPath := range dbs {
		items, err := hybridSearchDB(dbPath, query, queryVec, limit)
		if err != nil {
			return nil, err
		}
		allItems = append(allItems, items...)
	}

	// Global re-rank and trim.
	sort.Slice(allItems, func(i, j int) bool { return allItems[i].Score > allItems[j].Score })
	if limit > 0 && len(allItems) > limit {
		allItems = allItems[:limit]
	}
	if allItems == nil {
		allItems = []ResultItem{}
	}
	return &Result{Query: query, Results: allItems}, nil
}

func hybridSearchDB(dbPath, query string, queryVec []float32, limit int) ([]ResultItem, error) {
	store, err := storage.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("search: open %s: %w", dbPath, err)
	}
	defer store.Close()

	fetch := limit * 2
	if fetch < defaultLimit {
		fetch = defaultLimit
	}

	// FTS results.
	ftsRaw, err := store.Search(query, fetch)
	if err != nil {
		return nil, fmt.Errorf("search: fts %s: %w", dbPath, err)
	}

	if queryVec == nil {
		// No embedder: return FTS-only results.
		return toItems(ftsRaw), nil
	}

	// Vector results.
	vecRaw, err := store.SearchSimilar(queryVec, fetch)
	if err != nil {
		return nil, fmt.Errorf("search: similar %s: %w", dbPath, err)
	}

	return mergeResults(ftsRaw, vecRaw, limit), nil
}

// mergeResults combines FTS and vector results with a linear score blend.
// FTS scores (bm25, negative, lower=better) are normalized to [0,1].
// Vector scores (cosine, [0,1]) are used directly.
func mergeResults(fts, vec []storage.SearchResult, limit int) []ResultItem {
	type entry struct {
		r        storage.SearchResult
		ftsScore float64
		vecScore float64
	}
	byID := make(map[int64]*entry)

	// Collect FTS.
	var minFTS, maxFTS float64
	if len(fts) > 0 {
		minFTS, maxFTS = fts[0].Score, fts[0].Score
		for _, r := range fts[1:] {
			if r.Score < minFTS {
				minFTS = r.Score
			}
			if r.Score > maxFTS {
				maxFTS = r.Score
			}
		}
	}
	ftsSpan := maxFTS - minFTS

	for _, r := range fts {
		var norm float64
		if ftsSpan == 0 {
			norm = 1.0
		} else {
			norm = (maxFTS - r.Score) / ftsSpan
		}
		id := r.Symbol.ID
		if byID[id] == nil {
			byID[id] = &entry{r: r}
		}
		byID[id].ftsScore = norm
	}

	// Collect vector.
	for _, r := range vec {
		id := r.Symbol.ID
		if byID[id] == nil {
			byID[id] = &entry{r: r}
		}
		byID[id].vecScore = r.Score
	}

	// Compute combined score.
	items := make([]ResultItem, 0, len(byID))
	for _, e := range byID {
		combined := hybridFTSWeight*e.ftsScore + hybridVectorWeight*e.vecScore
		items = append(items, ResultItem{
			Name:      e.r.Name,
			Kind:      e.r.Kind,
			Signature: e.r.Signature,
			Doc:       e.r.Doc,
			File:      e.r.FilePath,
			LineStart: e.r.LineStart,
			LineEnd:   e.r.LineEnd,
			Score:     math.Round(combined*100) / 100,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

// listDBs returns all *.db paths in atlasDir, empty slice if dir absent.
func listDBs(atlasDir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(atlasDir, "*.db"))
	if err != nil {
		return nil, fmt.Errorf("search: glob %s: %w", atlasDir, err)
	}
	return matches, nil
}
