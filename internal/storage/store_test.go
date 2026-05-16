package storage

import (
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestUpsertAndGetRepo(t *testing.T) {
	s := openTestStore(t)

	tests := []struct {
		name string
		repo Repo
	}{
		{
			name: "insert new repo",
			repo: Repo{Path: "/tmp/myrepo", Name: "myrepo", Lang: "go", IndexedAt: time.Now().Unix()},
		},
		{
			name: "upsert same path updates fields",
			repo: Repo{Path: "/tmp/myrepo", Name: "myrepo-renamed", Lang: "go", IndexedAt: time.Now().Unix() + 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := s.UpsertRepo(tt.repo)
			if err != nil {
				t.Fatalf("UpsertRepo() error: %v", err)
			}
			if id == 0 {
				t.Error("UpsertRepo() returned id=0")
			}

			got, err := s.GetRepoByPath(tt.repo.Path)
			if err != nil {
				t.Fatalf("GetRepoByPath() error: %v", err)
			}
			if got == nil {
				t.Fatal("GetRepoByPath() returned nil")
			}
			if got.Name != tt.repo.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.repo.Name)
			}
		})
	}
}

func TestListRepos(t *testing.T) {
	s := openTestStore(t)

	repos := []Repo{
		{Path: "/tmp/a", Name: "a", Lang: "go", IndexedAt: 1},
		{Path: "/tmp/b", Name: "b", Lang: "c", IndexedAt: 2},
	}
	for _, r := range repos {
		if _, err := s.UpsertRepo(r); err != nil {
			t.Fatalf("UpsertRepo() error: %v", err)
		}
	}

	got, err := s.ListRepos()
	if err != nil {
		t.Fatalf("ListRepos() error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("ListRepos() returned %d repos, want 2", len(got))
	}
}

func TestUpsertAndGetFile(t *testing.T) {
	s := openTestStore(t)

	repoID, _ := s.UpsertRepo(Repo{Path: "/tmp/r", Name: "r", Lang: "go", IndexedAt: 1})

	file := File{RepoID: repoID, Path: "main.go", Hash: "abc123", Mtime: 1000}
	id, err := s.UpsertFile(file)
	if err != nil {
		t.Fatalf("UpsertFile() error: %v", err)
	}
	if id == 0 {
		t.Error("UpsertFile() returned id=0")
	}

	got, err := s.GetFile(repoID, "main.go")
	if err != nil {
		t.Fatalf("GetFile() error: %v", err)
	}
	if got == nil {
		t.Fatal("GetFile() returned nil")
	}
	if got.Hash != "abc123" {
		t.Errorf("Hash = %q, want %q", got.Hash, "abc123")
	}
}

func TestInsertSymbolsAndSearch(t *testing.T) {
	s := openTestStore(t)

	repoID, _ := s.UpsertRepo(Repo{Path: "/tmp/r", Name: "r", Lang: "go", IndexedAt: 1})
	fileID, _ := s.UpsertFile(File{RepoID: repoID, Path: "main.go", Hash: "x", Mtime: 1})

	symbols := []Symbol{
		{FileID: fileID, RepoID: repoID, Name: "CreateDevice", Kind: "function",
			Signature: "func CreateDevice(name string) *Device", Doc: "CreateDevice creates a GPU device.", LineStart: 10, LineEnd: 20},
		{FileID: fileID, RepoID: repoID, Name: "Device", Kind: "struct",
			Signature: "", Doc: "Device represents a GPU device.", LineStart: 5, LineEnd: 8},
	}

	if err := s.InsertSymbols(symbols); err != nil {
		t.Fatalf("InsertSymbols() error: %v", err)
	}

	tests := []struct {
		query   string
		wantMin int
	}{
		{query: "CreateDevice", wantMin: 1},
		{query: "GPU device", wantMin: 1},
		{query: "nonexistent_xyzzy", wantMin: 0},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			results, err := s.Search(tt.query, 10)
			if err != nil {
				t.Fatalf("Search(%q) error: %v", tt.query, err)
			}
			if len(results) < tt.wantMin {
				t.Errorf("Search(%q) returned %d results, want >= %d", tt.query, len(results), tt.wantMin)
			}
		})
	}
}

func TestDeleteSymbolsForFile(t *testing.T) {
	s := openTestStore(t)

	repoID, _ := s.UpsertRepo(Repo{Path: "/tmp/r", Name: "r", Lang: "go", IndexedAt: 1})
	fileID, _ := s.UpsertFile(File{RepoID: repoID, Path: "main.go", Hash: "x", Mtime: 1})

	symbols := []Symbol{
		{FileID: fileID, RepoID: repoID, Name: "Foo", Kind: "function", LineStart: 1, LineEnd: 5},
	}
	if err := s.InsertSymbols(symbols); err != nil {
		t.Fatalf("InsertSymbols() error: %v", err)
	}

	if err := s.DeleteSymbolsForFile(fileID); err != nil {
		t.Fatalf("DeleteSymbolsForFile() error: %v", err)
	}

	results, err := s.Search("Foo", 10)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search() returned %d results after delete, want 0", len(results))
	}
}
