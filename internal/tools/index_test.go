package tools

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/brainer.sh/atlas/internal/storage"
)

func openTestStore(t *testing.T) *storage.Store {
	t.Helper()
	s, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func fixtureDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "go", "simple")
}

func fixtureDirC() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "c", "simple")
}

func fixtureDirCpp() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "cpp", "simple")
}

func TestIndexRepo(t *testing.T) {
	store := openTestStore(t)
	dir := fixtureDir()

	result, err := IndexRepo(dir, store)
	if err != nil {
		t.Fatalf("IndexRepo() error: %v", err)
	}

	if result.FilesIndexed == 0 {
		t.Error("FilesIndexed = 0, want > 0")
	}
	if result.SymbolsIndexed == 0 {
		t.Error("SymbolsIndexed = 0, want > 0")
	}
	if result.Repo == "" {
		t.Error("Repo is empty")
	}
}

func TestIndexRepo_SearchAfterIndex(t *testing.T) {
	store := openTestStore(t)
	dir := fixtureDir()

	if _, err := IndexRepo(dir, store); err != nil {
		t.Fatalf("IndexRepo() error: %v", err)
	}

	tests := []struct {
		query   string
		wantMin int
	}{
		{query: "Add", wantMin: 1},
		{query: "Greeter", wantMin: 1},
		{query: "greeting", wantMin: 1},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			results, err := store.Search(tt.query, 10)
			if err != nil {
				t.Fatalf("Search(%q) error: %v", tt.query, err)
			}
			if len(results) < tt.wantMin {
				t.Errorf("Search(%q) = %d results, want >= %d", tt.query, len(results), tt.wantMin)
			}
		})
	}
}

func TestReindexRepo_NoChanges(t *testing.T) {
	store := openTestStore(t)
	dir := fixtureDir()

	first, err := IndexRepo(dir, store)
	if err != nil {
		t.Fatalf("IndexRepo() error: %v", err)
	}

	second, err := ReindexRepo(dir, store)
	if err != nil {
		t.Fatalf("ReindexRepo() error: %v", err)
	}

	if second.FilesIndexed != 0 {
		t.Errorf("ReindexRepo with no changes: FilesIndexed = %d, want 0", second.FilesIndexed)
	}
	_ = first
}

func TestDetectLang(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{name: "go", dir: fixtureDir(), want: "go"},
		{name: "c", dir: fixtureDirC(), want: "c"},
		{name: "cpp", dir: fixtureDirCpp(), want: "cpp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectLang(tt.dir)
			if got != tt.want {
				t.Errorf("detectLang = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIndexRepo_C(t *testing.T) {
	store := openTestStore(t)
	result, err := IndexRepo(fixtureDirC(), store)
	if err != nil {
		t.Fatalf("IndexRepo(C) error: %v", err)
	}
	if result.FilesIndexed == 0 {
		t.Error("FilesIndexed = 0")
	}
	if result.SymbolsIndexed == 0 {
		t.Error("SymbolsIndexed = 0")
	}

	results, err := store.Search("add", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected symbol 'add' in search results")
	}
}

func TestIndexRepo_Cpp(t *testing.T) {
	store := openTestStore(t)
	result, err := IndexRepo(fixtureDirCpp(), store)
	if err != nil {
		t.Fatalf("IndexRepo(C++) error: %v", err)
	}
	if result.FilesIndexed == 0 {
		t.Error("FilesIndexed = 0")
	}
	if result.SymbolsIndexed == 0 {
		t.Error("SymbolsIndexed = 0")
	}

	results, err := store.Search("Shape", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected symbol 'Shape' in search results")
	}
}

func TestReindexRepo_NeverIndexed(t *testing.T) {
	store := openTestStore(t)
	dir := fixtureDir()

	result, err := ReindexRepo(dir, store)
	if err != nil {
		t.Fatalf("ReindexRepo() on fresh store error: %v", err)
	}
	if result.FilesIndexed == 0 {
		t.Error("ReindexRepo on fresh store: FilesIndexed = 0, want > 0")
	}
}
