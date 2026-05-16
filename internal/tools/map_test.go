package tools

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/brainer.sh/atlas/internal/storage"
)

func seedMapDB(t *testing.T, atlasDir string) {
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

func TestGetMap_noAtlasDir(t *testing.T) {
	result, err := GetMap("/nonexistent/atlas", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Diagram == "" {
		t.Error("diagram is empty")
	}
}

func TestGetMap_globalContainsMermaidHeader(t *testing.T) {
	atlasDir := t.TempDir()
	seedMapDB(t, atlasDir)

	result, err := GetMap(atlasDir, "")
	if err != nil {
		t.Fatalf("GetMap: %v", err)
	}
	if !strings.HasPrefix(result.Diagram, "graph TD") {
		t.Errorf("diagram does not start with 'graph TD': %s", result.Diagram)
	}
	if result.Focus != "" {
		t.Errorf("focus = %q, want empty", result.Focus)
	}
}

func TestGetMap_globalContainsRepoName(t *testing.T) {
	atlasDir := t.TempDir()
	seedMapDB(t, atlasDir)

	result, err := GetMap(atlasDir, "")
	if err != nil {
		t.Fatalf("GetMap: %v", err)
	}
	if !strings.Contains(result.Diagram, "simple") {
		t.Errorf("diagram missing repo name 'simple': %s", result.Diagram)
	}
}

func TestGetMap_focusedContainsSymbol(t *testing.T) {
	atlasDir := t.TempDir()
	seedMapDB(t, atlasDir)

	result, err := GetMap(atlasDir, "Add")
	if err != nil {
		t.Fatalf("GetMap focused: %v", err)
	}
	if !strings.HasPrefix(result.Diagram, "graph TD") {
		t.Errorf("diagram does not start with 'graph TD': %s", result.Diagram)
	}
	if result.Focus != "Add" {
		t.Errorf("focus = %q, want %q", result.Focus, "Add")
	}
	if !strings.Contains(result.Diagram, "Add") {
		t.Errorf("diagram missing symbol 'Add': %s", result.Diagram)
	}
}

func TestGetMap_focusedContainsSiblings(t *testing.T) {
	atlasDir := t.TempDir()
	seedMapDB(t, atlasDir)

	result, err := GetMap(atlasDir, "Add")
	if err != nil {
		t.Fatalf("GetMap focused: %v", err)
	}
	// Fixture has Add, Greeter, Greet, Sayer in same file.
	for _, sym := range []string{"Greeter", "Greet"} {
		if !strings.Contains(result.Diagram, sym) {
			t.Errorf("diagram missing sibling %q: %s", sym, result.Diagram)
		}
	}
}

func TestGetMap_focusedNotFound(t *testing.T) {
	atlasDir := t.TempDir()
	seedMapDB(t, atlasDir)

	result, err := GetMap(atlasDir, "NonExistentXYZ")
	if err != nil {
		t.Fatalf("GetMap: %v", err)
	}
	if result.Focus != "NonExistentXYZ" {
		t.Errorf("focus = %q, want %q", result.Focus, "NonExistentXYZ")
	}
	if result.Diagram != "graph TD" {
		t.Errorf("expected empty graph, got: %s", result.Diagram)
	}
}
