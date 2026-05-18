package search

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/brainer.sh/atlas/internal/embeddings"
	"github.com/brainer.sh/atlas/internal/storage"
	"github.com/brainer.sh/atlas/internal/tools"
)

// TestSemanticSearch_EndToEnd seeds a DB, embeds all symbols with FakeEmbedder,
// then runs HybridSearch and verifies results are returned with valid scores.
func TestSemanticSearch_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "repo.db")

	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	repoID, _ := store.UpsertRepo(storage.Repo{Path: "/tmp/r", Name: "r", Lang: "go", IndexedAt: 1})
	fileID, _ := store.UpsertFile(storage.File{RepoID: repoID, Path: "gpu.go", Hash: "x", Mtime: 1})
	syms := []storage.Symbol{
		{FileID: fileID, RepoID: repoID, Name: "CreateDevice", Kind: "function",
			Signature: "func CreateDevice() *Device", Doc: "Creates a GPU device.", LineStart: 1, LineEnd: 10},
		{FileID: fileID, RepoID: repoID, Name: "DestroyDevice", Kind: "function",
			Signature: "func DestroyDevice(d *Device)", Doc: "Destroys a GPU device.", LineStart: 12, LineEnd: 20},
		{FileID: fileID, RepoID: repoID, Name: "ListDevices", Kind: "function",
			Signature: "func ListDevices() []*Device", Doc: "Lists all GPU devices.", LineStart: 22, LineEnd: 30},
	}
	if err := store.InsertSymbols(syms); err != nil {
		t.Fatalf("InsertSymbols: %v", err)
	}

	e := embeddings.FakeEmbedder{}
	if err := tools.EmbedAll(context.Background(), store, e); err != nil {
		t.Fatalf("EmbedAll: %v", err)
	}
	store.Close()

	result, err := HybridSearch(context.Background(), dir, "CreateDevice", e, 10)
	if err != nil {
		t.Fatalf("HybridSearch: %v", err)
	}

	if len(result.Results) == 0 {
		t.Fatal("expected results, got 0")
	}
	if result.Query != "CreateDevice" {
		t.Errorf("Query = %q, want CreateDevice", result.Query)
	}

	seen := make(map[string]bool)
	for _, r := range result.Results {
		seen[r.Name] = true
		if r.Score < 0 || r.Score > 1 {
			t.Errorf("%s: Score = %v, want [0,1]", r.Name, r.Score)
		}
	}
	if !seen["CreateDevice"] {
		t.Error("CreateDevice not in results")
	}
}
