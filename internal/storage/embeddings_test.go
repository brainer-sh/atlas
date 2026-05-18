package storage

import (
	"testing"
)

func seedSymbol(t *testing.T, s *Store) int64 {
	t.Helper()
	repoID, err := s.UpsertRepo(Repo{Path: "/tmp/testrepo", Name: "testrepo", Lang: "go", IndexedAt: 1})
	if err != nil {
		t.Fatalf("UpsertRepo: %v", err)
	}
	fileID, err := s.UpsertFile(File{RepoID: repoID, Path: "main.go", Hash: "abc", Mtime: 1})
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}
	syms := []Symbol{{FileID: fileID, RepoID: repoID, Name: "main", Kind: "function", LineStart: 1, LineEnd: 5}}
	if err := s.InsertSymbols(syms); err != nil {
		t.Fatalf("InsertSymbols: %v", err)
	}
	all, err := s.GetAllSymbols()
	if err != nil {
		t.Fatalf("GetAllSymbols: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("no symbols inserted")
	}
	return all[0].ID
}

func TestEncodeDecodeVec(t *testing.T) {
	tests := []struct {
		name string
		vec  []float32
	}{
		{name: "zero", vec: []float32{0, 0, 0, 0}},
		{name: "unit", vec: []float32{1, 0, 0, 0}},
		{name: "floats", vec: []float32{0.1, 0.2, 0.3, -0.4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeVec(encodeVec(tt.vec))
			if len(got) != len(tt.vec) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.vec))
			}
			for i := range tt.vec {
				if got[i] != tt.vec[i] {
					t.Errorf("[%d] = %v, want %v", i, got[i], tt.vec[i])
				}
			}
		})
	}
}

func TestCosine(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float64
	}{
		{name: "identical", a: []float32{1, 0, 0}, b: []float32{1, 0, 0}, want: 1.0},
		{name: "orthogonal", a: []float32{1, 0, 0}, b: []float32{0, 1, 0}, want: 0.0},
		{name: "dim mismatch", a: []float32{1, 0}, b: []float32{1, 0, 0}, want: 0.0},
		{name: "zero vector", a: []float32{0, 0, 0}, b: []float32{1, 0, 0}, want: 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosine(tt.a, tt.b)
			if diff := got - tt.want; diff < -1e-6 || diff > 1e-6 {
				t.Errorf("cosine = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStoreAndSearchSimilar(t *testing.T) {
	s := openTestStore(t)
	id := seedSymbol(t, s)

	vec := []float32{0.5, 0.5, 0.5, 0.5}
	if err := s.StoreEmbedding(id, vec); err != nil {
		t.Fatalf("StoreEmbedding: %v", err)
	}

	results, err := s.SearchSimilar(vec, 5)
	if err != nil {
		t.Fatalf("SearchSimilar: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Symbol.ID != id {
		t.Errorf("ID = %d, want %d", results[0].Symbol.ID, id)
	}
	if results[0].Score < 0.99 {
		t.Errorf("Score = %v, want ~1.0 (same vector)", results[0].Score)
	}
}

func TestStoreEmbedding_Replace(t *testing.T) {
	s := openTestStore(t)
	id := seedSymbol(t, s)

	v1 := []float32{1, 0, 0, 0}
	v2 := []float32{0, 1, 0, 0}

	if err := s.StoreEmbedding(id, v1); err != nil {
		t.Fatalf("first StoreEmbedding: %v", err)
	}
	if err := s.StoreEmbedding(id, v2); err != nil {
		t.Fatalf("second StoreEmbedding: %v", err)
	}

	// Searching with v2 should score ~1.0 (v2 was stored last).
	results, err := s.SearchSimilar(v2, 1)
	if err != nil {
		t.Fatalf("SearchSimilar: %v", err)
	}
	if len(results) != 1 || results[0].Score < 0.99 {
		t.Errorf("expected score ~1.0 after replace, got %v", results)
	}
}

func TestGetAllSymbols_Empty(t *testing.T) {
	s := openTestStore(t)
	syms, err := s.GetAllSymbols()
	if err != nil {
		t.Fatalf("GetAllSymbols: %v", err)
	}
	if len(syms) != 0 {
		t.Errorf("expected 0 symbols, got %d", len(syms))
	}
}
