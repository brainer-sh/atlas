package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brainer.sh/atlas/internal/mcp"
	"github.com/brainer.sh/atlas/internal/search"
	"github.com/brainer.sh/atlas/internal/storage"
	"github.com/brainer.sh/atlas/internal/tools"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "index":
		if len(os.Args) < 3 {
			fatalf("usage: atlas index <path>")
		}
		cmdIndex(os.Args[2])
	case "reindex":
		if len(os.Args) < 3 {
			fatalf("usage: atlas reindex <path>")
		}
		cmdReindex(os.Args[2])
	case "list":
		cmdList()
	case "serve":
		cmdServe()
	case "search":
		if len(os.Args) < 3 {
			fatalf("usage: atlas search <query>")
		}
		cmdSearch(os.Args[2])
	case "--help", "-h", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "atlas: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func cmdIndex(repoPath string) {
	store, err := openStoreForRepo(repoPath)
	if err != nil {
		fatalf("index: %v", err)
	}
	defer store.Close()

	result, err := tools.IndexRepo(repoPath, store)
	if err != nil {
		fatalf("index: %v", err)
	}
	printJSON(map[string]any{
		"repo":            result.Repo,
		"path":            result.Path,
		"files_indexed":   result.FilesIndexed,
		"symbols_indexed": result.SymbolsIndexed,
		"duration_ms":     result.DurationMs,
	})
}

func cmdReindex(repoPath string) {
	store, err := openStoreForRepo(repoPath)
	if err != nil {
		fatalf("reindex: %v", err)
	}
	defer store.Close()

	result, err := tools.ReindexRepo(repoPath, store)
	if err != nil {
		fatalf("reindex: %v", err)
	}
	printJSON(map[string]any{
		"repo":            result.Repo,
		"path":            result.Path,
		"files_indexed":   result.FilesIndexed,
		"symbols_indexed": result.SymbolsIndexed,
		"duration_ms":     result.DurationMs,
	})
}

func cmdList() {
	dir, err := atlasDir()
	if err != nil {
		fatalf("list: %v", err)
	}
	repos, err := tools.ListRepos(dir)
	if err != nil {
		fatalf("list: %v", err)
	}
	if repos == nil {
		repos = []tools.RepoEntry{}
	}
	printJSON(map[string]any{"repos": repos})
}

func cmdServe() {
	s := mcp.New()
	if err := mcp.Serve(s); err != nil {
		fatalf("serve: %v", err)
	}
}

func cmdSearch(query string) {
	dir, err := atlasDir()
	if err != nil {
		fatalf("search: %v", err)
	}
	result, err := search.Search(dir, query)
	if err != nil {
		fatalf("search: %v", err)
	}
	printJSON(result)
}

func openStoreForRepo(repoPath string) (*storage.Store, error) {
	dir, err := atlasDir()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dir, filepath.Base(repoPath)+".db")
	return storage.Open(dbPath)
}

func atlasDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	dir := filepath.Join(home, ".atlas")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create ~/.atlas: %w", err)
	}
	return dir, nil
}

func printJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatalf("marshal: %v", err)
	}
	fmt.Println(string(b))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "atlas: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, `Atlas - codebase indexer for AI agents

Usage:
  atlas index <path>     index a repository
  atlas reindex <path>   re-index modified files only
  atlas list             list all indexed repositories
  atlas serve            start the MCP server (stdio)
  atlas search <query>   search symbols (debug)
  atlas --help           show this help
`)
}
