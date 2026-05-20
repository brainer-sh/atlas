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

func TestExploreSymbol_callersCallees(t *testing.T) {
	atlasDir := t.TempDir()
	seedExploreDB(t, atlasDir)

	// Run() calls Add(), so Add should have Run as a caller.
	add, err := ExploreSymbol(atlasDir, "Add")
	if err != nil {
		t.Fatalf("ExploreSymbol(Add): %v", err)
	}
	if add == nil {
		t.Fatal("ExploreSymbol(Add) = nil")
	}
	if len(add.Callers) == 0 {
		t.Error("Add.Callers is empty, want at least [Run]")
	}
	found := false
	for _, c := range add.Callers {
		if c == "Run" {
			found = true
		}
	}
	if !found {
		t.Errorf("Add.Callers = %v, want to contain Run", add.Callers)
	}

	// Run() calls Add(), so Run should have Add as a callee.
	run, err := ExploreSymbol(atlasDir, "Run")
	if err != nil {
		t.Fatalf("ExploreSymbol(Run): %v", err)
	}
	if run == nil {
		t.Fatal("ExploreSymbol(Run) = nil")
	}
	if len(run.Callees) == 0 {
		t.Error("Run.Callees is empty, want at least [Add]")
	}
	foundCallee := false
	for _, c := range run.Callees {
		if c == "Add" {
			foundCallee = true
		}
	}
	if !foundCallee {
		t.Errorf("Run.Callees = %v, want to contain Add", run.Callees)
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
