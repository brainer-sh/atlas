package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brainer.sh/atlas/internal/storage"
)

// MapResult is the response for the get_map tool.
type MapResult struct {
	Focus   string `json:"focus"`
	Diagram string `json:"diagram"`
}

// GetMap returns a Mermaid diagram of the global architecture (focus == "")
// or a diagram centered on a specific symbol (focus == symbol name).
func GetMap(atlasDir, focus string) (*MapResult, error) {
	if focus != "" {
		return getFocusedMap(atlasDir, focus)
	}
	return getGlobalMap(atlasDir)
}

// getGlobalMap builds a Mermaid diagram showing the package directory structure
// for all indexed repos found in atlasDir.
func getGlobalMap(atlasDir string) (*MapResult, error) {
	entries, err := os.ReadDir(atlasDir)
	if os.IsNotExist(err) {
		return &MapResult{Diagram: "graph TD"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tools/map: read dir %s: %w", atlasDir, err)
	}

	var lines []string
	lines = append(lines, "graph TD")

	nodeIdx := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".db" {
			continue
		}
		store, err := storage.Open(filepath.Join(atlasDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("tools/map: open %s: %w", e.Name(), err)
		}
		dbLines, n, err := globalLinesFromDB(store, nodeIdx)
		store.Close()
		if err != nil {
			return nil, err
		}
		lines = append(lines, dbLines...)
		nodeIdx += n
	}

	return &MapResult{Diagram: strings.Join(lines, "\n")}, nil
}

func globalLinesFromDB(store *storage.Store, startIdx int) ([]string, int, error) {
	files, err := store.ListAllFiles()
	if err != nil {
		return nil, 0, fmt.Errorf("tools/map: list files: %w", err)
	}
	if len(files) == 0 {
		return nil, 0, nil
	}

	// Group unique dirs per repo.
	type repoInfo struct {
		name string
		path string
		dirs map[string]struct{}
	}
	repos := map[int64]*repoInfo{}
	repoOrder := []int64{}

	for _, f := range files {
		if _, ok := repos[f.RepoID]; !ok {
			repos[f.RepoID] = &repoInfo{
				name: f.RepoName,
				path: f.RepoPath,
				dirs: map[string]struct{}{},
			}
			repoOrder = append(repoOrder, f.RepoID)
		}
		dir := filepath.Dir(f.FilePath)
		if dir == "." {
			dir = f.RepoName
		}
		repos[f.RepoID].dirs[dir] = struct{}{}
	}

	var lines []string
	idx := startIdx
	for _, repoID := range repoOrder {
		r := repos[repoID]
		safeRepo := mermaidID(r.name)
		lines = append(lines, fmt.Sprintf("  subgraph %s[\"%s\"]", safeRepo, r.name))

		dirs := make([]string, 0, len(r.dirs))
		for d := range r.dirs {
			dirs = append(dirs, d)
		}
		sort.Strings(dirs)

		for _, d := range dirs {
			nodeID := fmt.Sprintf("n%d", idx)
			lines = append(lines, fmt.Sprintf("    %s[\"%s\"]", nodeID, d))
			idx++
		}
		lines = append(lines, "  end")
	}
	return lines, idx - startIdx, nil
}

// getFocusedMap builds a Mermaid diagram showing a symbol and all sibling
// symbols in the same file.
func getFocusedMap(atlasDir, focus string) (*MapResult, error) {
	entries, err := os.ReadDir(atlasDir)
	if os.IsNotExist(err) {
		return &MapResult{Focus: focus, Diagram: "graph TD"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tools/map: read dir %s: %w", atlasDir, err)
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".db" {
			continue
		}
		store, err := storage.Open(filepath.Join(atlasDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("tools/map: open %s: %w", e.Name(), err)
		}
		diagram, err := focusedDiagramFromDB(store, focus)
		store.Close()
		if err != nil {
			return nil, err
		}
		if diagram != "" {
			return &MapResult{Focus: focus, Diagram: diagram}, nil
		}
	}

	// Symbol not found - return an empty graph.
	return &MapResult{Focus: focus, Diagram: "graph TD"}, nil
}

func focusedDiagramFromDB(store *storage.Store, focus string) (string, error) {
	detail, err := store.GetSymbolByName(focus)
	if err != nil {
		return "", fmt.Errorf("tools/map: get symbol: %w", err)
	}
	if detail == nil {
		return "", nil
	}

	siblings, err := store.GetSymbolsByFileID(detail.FileID)
	if err != nil {
		return "", fmt.Errorf("tools/map: get siblings: %w", err)
	}

	var lines []string
	lines = append(lines, "graph TD")
	fileLabel := filepath.Base(detail.FilePath)
	lines = append(lines, fmt.Sprintf("  subgraph file[\"%s\"]", fileLabel))

	for _, s := range siblings {
		nodeID := mermaidID(s.Name)
		if s.Name == focus {
			lines = append(lines, fmt.Sprintf("    %s[\"%s\\n%s\"]:::focus", nodeID, s.Name, s.Kind))
		} else {
			lines = append(lines, fmt.Sprintf("    %s[\"%s\\n%s\"]", nodeID, s.Name, s.Kind))
		}
	}
	lines = append(lines, "  end")
	lines = append(lines, "  classDef focus fill:#f90,color:#000")

	return strings.Join(lines, "\n"), nil
}

// mermaidID converts an arbitrary string to a safe Mermaid node identifier.
func mermaidID(s string) string {
	r := strings.NewReplacer("/", "_", ".", "_", "-", "_", " ", "_")
	return r.Replace(s)
}
