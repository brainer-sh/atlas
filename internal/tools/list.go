package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/brainer.sh/atlas/internal/storage"
)

// RepoEntry is a single repo returned by ListRepos.
type RepoEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Lang      string `json:"lang"`
	IndexedAt string `json:"indexed_at"` // RFC 3339
	Files     int64  `json:"files"`
	Symbols   int64  `json:"symbols"`
}

// ListRepos scans atlasDir for *.db files and returns all indexed repos.
func ListRepos(atlasDir string) ([]RepoEntry, error) {
	entries, err := os.ReadDir(atlasDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tools/list: read dir %s: %w", atlasDir, err)
	}

	var repos []RepoEntry
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".db" {
			continue
		}
		dbPath := filepath.Join(atlasDir, e.Name())
		batch, err := reposFromDB(dbPath)
		if err != nil {
			return nil, fmt.Errorf("tools/list: read %s: %w", dbPath, err)
		}
		repos = append(repos, batch...)
	}
	return repos, nil
}

func reposFromDB(dbPath string) ([]RepoEntry, error) {
	store, err := storage.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	stats, err := store.ListRepos()
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}

	entries := make([]RepoEntry, len(stats))
	for i, s := range stats {
		entries[i] = RepoEntry{
			Name:      s.Name,
			Path:      s.Path,
			Lang:      s.Lang,
			IndexedAt: time.Unix(s.IndexedAt, 0).UTC().Format(time.RFC3339),
			Files:     s.Files,
			Symbols:   s.Symbols,
		}
	}
	return entries, nil
}
