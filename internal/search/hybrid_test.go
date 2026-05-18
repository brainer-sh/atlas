package search

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/brainer.sh/atlas/internal/embeddings"
	"github.com/brainer.sh/atlas/internal/storage"
)

func seedVecDB(t *testing.T, dir string) string {
	t.Helper()
	dbPath := filepath.Join(dir, "testrepo.db")
	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	repoID, _ := store.UpsertRepo(storage.Repo{Path: "/tmp/r", Name: "r", Lang: "go", IndexedAt: 1})
	fileID, _ := store.UpsertFile(storage.File{RepoID: repoID, Path: "main.go", Hash: "x", Mtime: 1})
	syms := []storage.Symbol{
		{FileID: fileID, RepoID: repoID, Name: "CreateDevice", Kind: "function",
			Signature: "func CreateDevice() *Device", Doc: "Creates a GPU device.", LineStart: 1, LineEnd: 10},
		{FileID: fileID, RepoID: repoID, Name: "DestroyDevice", Kind: "function",
			Signature: "func DestroyDevice(d *Device)", Doc: "Destroys a GPU device.", LineStart: 12, LineEnd: 20},
	}
	if err := store.InsertSymbols(syms); err != nil {
		t.Fatalf("InsertSymbols: %v", err)
	}
	return dbPath
}

func TestHybridSearch_NoEmbedder(t *testing.T) {
	dir := t.TempDir()
	seedVecDB(t, dir)

	result, err := HybridSearch(context.Background(), dir, "CreateDevice", nil, 10)
	if err != nil {
		t.Fatalf("HybridSearch: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected at least one result, got 0")
	}
	if result.Results[0].Name != "CreateDevice" {
		t.Errorf("top result = %q, want CreateDevice", result.Results[0].Name)
	}
}

func TestHybridSearch_WithFakeEmbedder(t *testing.T) {
	dir := t.TempDir()
	dbPath := seedVecDB(t, dir)
	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	e := embeddings.FakeEmbedder{}
	syms, err := store.GetAllSymbols()
	if err != nil {
		t.Fatalf("GetAllSymbols: %v", err)
	}
	texts := make([]string, len(syms))
	for i, s := range syms {
		texts[i] = s.Kind + " " + s.Name + " " + s.Signature + " " + s.Doc
	}
	vecs, err := e.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	for i, s := range syms {
		if err := store.StoreEmbedding(s.ID, vecs[i]); err != nil {
			t.Fatalf("StoreEmbedding: %v", err)
		}
	}
	store.Close()

	result, err := HybridSearch(context.Background(), dir, "GPU device", e, 10)
	if err != nil {
		t.Fatalf("HybridSearch: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected results, got 0")
	}
	// All scores must be in [0, 1].
	for _, r := range result.Results {
		if r.Score < 0 || r.Score > 1 {
			t.Errorf("Score = %v out of [0,1]", r.Score)
		}
	}
}

func TestHybridSearch_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	result, err := HybridSearch(context.Background(), dir, "anything", nil, 10)
	if err != nil {
		t.Fatalf("HybridSearch: %v", err)
	}
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results for empty dir, got %d", len(result.Results))
	}
}

func TestMergeResults_FTSOnly(t *testing.T) {
	fts := []storage.SearchResult{
		{Symbol: storage.Symbol{ID: 1, Name: "Foo", Kind: "function"}, Score: -1.0},
		{Symbol: storage.Symbol{ID: 2, Name: "Bar", Kind: "function"}, Score: -0.5},
	}
	items := mergeResults(fts, nil, 10)
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}
	// Foo has better FTS score (more negative), should rank higher.
	if items[0].Name != "Foo" {
		t.Errorf("top = %q, want Foo", items[0].Name)
	}
}
