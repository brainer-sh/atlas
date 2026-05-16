package tools

import (
	"path/filepath"
	"testing"

	"github.com/brainer.sh/atlas/internal/storage"
)

func seedExploreDB(t *testing.T, atlasDir string) {
	t.Helper()
	store, err := storage.Open(filepath.Join(atlasDir, "simple.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	_, err = IndexRepo(fixtureDir(), store)
	store.Close()
	if err != nil {
		t.Fatalf("IndexRepo: %v", err)
	}
}

func TestExploreSymbol_notFound(t *testing.T) {
	atlasDir := t.TempDir()
	seedExploreDB(t, atlasDir)

	result, err := ExploreSymbol(atlasDir, "NonExistentXYZ")
	if err != nil {
		t.Fatalf("ExploreSymbol: %v", err)
	}
	if result != nil {
		t.Errorf("want nil, got %+v", result)
	}
}

func TestExploreSymbol_noAtlasDir(t *testing.T) {
	result, err := ExploreSymbol("/nonexistent/path/atlas", "Add")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("want nil, got %+v", result)
	}
}

func TestExploreSymbol_function(t *testing.T) {
	atlasDir := t.TempDir()
	seedExploreDB(t, atlasDir)

	result, err := ExploreSymbol(atlasDir, "Add")
	if err != nil {
		t.Fatalf("ExploreSymbol: %v", err)
	}
	if result == nil {
		t.Fatal("want result, got nil")
	}

	tests := []struct {
		field string
		got   any
		want  any
	}{
		{"symbol", result.Symbol, "Add"},
		{"kind", result.Kind, "function"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("result.%s = %v, want %v", tt.field, tt.got, tt.want)
		}
	}
	if result.File == "" {
		t.Error("file is empty")
	}
	if result.LineStart <= 0 {
		t.Errorf("line_start = %d, want > 0", result.LineStart)
	}
	if result.LineEnd < result.LineStart {
		t.Errorf("line_end %d < line_start %d", result.LineEnd, result.LineStart)
	}
	if result.Callers == nil {
		t.Error("callers is nil, want empty slice")
	}
	if result.Callees == nil {
		t.Error("callees is nil, want empty slice")
	}
}

func TestExploreSymbol_struct(t *testing.T) {
	atlasDir := t.TempDir()
	seedExploreDB(t, atlasDir)

	result, err := ExploreSymbol(atlasDir, "Greeter")
	if err != nil {
		t.Fatalf("ExploreSymbol: %v", err)
	}
	if result == nil {
		t.Fatal("want result, got nil")
	}
	if result.Symbol != "Greeter" {
		t.Errorf("symbol = %q, want %q", result.Symbol, "Greeter")
	}
}
