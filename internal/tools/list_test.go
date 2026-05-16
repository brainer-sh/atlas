package tools

import (
	"path/filepath"
	"testing"

	"github.com/brainer.sh/atlas/internal/storage"
)

func TestListRepos_empty(t *testing.T) {
	atlasDir := t.TempDir()
	repos, err := ListRepos(atlasDir)
	if err != nil {
		t.Fatalf("ListRepos empty dir: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("want 0 repos, got %d", len(repos))
	}
}

func TestListRepos_notExist(t *testing.T) {
	repos, err := ListRepos("/nonexistent/path/atlas")
	if err != nil {
		t.Fatalf("ListRepos nonexistent dir: %v", err)
	}
	if repos != nil {
		t.Errorf("want nil, got %v", repos)
	}
}

func TestListRepos_afterIndex(t *testing.T) {
	atlasDir := t.TempDir()
	fixture := fixtureDir()

	// Seed a db the same way the MCP handler would.
	dbPath := filepath.Join(atlasDir, filepath.Base(fixture)+".db")
	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	result, err := IndexRepo(fixture, store)
	store.Close()
	if err != nil {
		t.Fatalf("IndexRepo: %v", err)
	}
	if result.FilesIndexed == 0 {
		t.Fatal("no files indexed, fixture may be empty")
	}

	repos, err := ListRepos(atlasDir)
	if err != nil {
		t.Fatalf("ListRepos: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("want 1 repo, got %d", len(repos))
	}

	got := repos[0]
	tests := []struct {
		field string
		got   any
		want  any
	}{
		{"name", got.Name, filepath.Base(fixture)},
		{"path", got.Path, fixture},
		{"lang", got.Lang, "go"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("repo.%s = %v, want %v", tt.field, tt.got, tt.want)
		}
	}
	if got.Files <= 0 {
		t.Errorf("repo.files = %d, want > 0", got.Files)
	}
	if got.Symbols <= 0 {
		t.Errorf("repo.symbols = %d, want > 0", got.Symbols)
	}
	if got.IndexedAt == "" {
		t.Error("repo.indexed_at is empty")
	}
}

func TestListRepos_multipleDBs(t *testing.T) {
	atlasDir := t.TempDir()
	fixture := fixtureDir()

	// Two dbs with different names simulate two indexed repos.
	for _, dbName := range []string{"repoA.db", "repoB.db"} {
		store, err := storage.Open(filepath.Join(atlasDir, dbName))
		if err != nil {
			t.Fatalf("open store %s: %v", dbName, err)
		}
		_, err = IndexRepo(fixture, store)
		store.Close()
		if err != nil {
			t.Fatalf("IndexRepo %s: %v", dbName, err)
		}
	}

	repos, err := ListRepos(atlasDir)
	if err != nil {
		t.Fatalf("ListRepos: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("want 2 repos, got %d", len(repos))
	}
}
