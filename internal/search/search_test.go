package search

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/brainer.sh/atlas/internal/storage"
	"github.com/brainer.sh/atlas/internal/tools"
)

func fixtureDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "go", "simple")
}

func seedDB(t *testing.T, atlasDir, dbName string) {
	t.Helper()
	store, err := storage.Open(filepath.Join(atlasDir, dbName))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	_, err = tools.IndexRepo(fixtureDir(), store)
	store.Close()
	if err != nil {
		t.Fatalf("IndexRepo: %v", err)
	}
}

func TestSearch_noAtlasDir(t *testing.T) {
	result, err := Search("/nonexistent/path/atlas", "anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Query != "anything" {
		t.Errorf("query = %q, want %q", result.Query, "anything")
	}
	if len(result.Results) != 0 {
		t.Errorf("want 0 results, got %d", len(result.Results))
	}
}

func TestSearch_emptyDir(t *testing.T) {
	result, err := Search(t.TempDir(), "anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 0 {
		t.Errorf("want 0 results, got %d", len(result.Results))
	}
}

func TestSearch_hitsSymbol(t *testing.T) {
	atlasDir := t.TempDir()
	seedDB(t, atlasDir, "simple.db")

	// The fixture defines an Add function; searching "Add" must return it.
	result, err := Search(atlasDir, "Add")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("want at least 1 result, got 0")
	}

	found := false
	for _, r := range result.Results {
		if r.Name == "Add" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("symbol Add not found in results: %+v", result.Results)
	}
}

func TestSearch_scoresNormalized(t *testing.T) {
	atlasDir := t.TempDir()
	seedDB(t, atlasDir, "simple.db")

	result, err := Search(atlasDir, "Add")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, r := range result.Results {
		if r.Score < 0 || r.Score > 1 {
			t.Errorf("score %v out of [0,1] for symbol %s", r.Score, r.Name)
		}
	}
}

func TestSearch_noMatch(t *testing.T) {
	atlasDir := t.TempDir()
	seedDB(t, atlasDir, "simple.db")

	result, err := Search(atlasDir, "zzznomatchzzz")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(result.Results) != 0 {
		t.Errorf("want 0 results, got %d", len(result.Results))
	}
}

func TestSearch_resultFields(t *testing.T) {
	atlasDir := t.TempDir()
	seedDB(t, atlasDir, "simple.db")

	result, err := Search(atlasDir, "Add")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("want results")
	}
	r := result.Results[0]
	if r.Name == "" {
		t.Error("name is empty")
	}
	if r.Kind == "" {
		t.Error("kind is empty")
	}
	if r.File == "" {
		t.Error("file is empty")
	}
	if r.LineStart <= 0 {
		t.Errorf("line_start = %d, want > 0", r.LineStart)
	}
}
