package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/brainer.sh/atlas/internal/search"
	"github.com/brainer.sh/atlas/internal/storage"
	"github.com/brainer.sh/atlas/internal/tools"
)

func registerTools(s *mcpserver.MCPServer) {
	s.AddTool(mcplib.NewTool("index_repo",
		mcplib.WithDescription("Index a repository for the first time."),
		mcplib.WithString("path",
			mcplib.Required(),
			mcplib.Description("Absolute path to the repository root."),
		),
	), handleIndexRepo)

	s.AddTool(mcplib.NewTool("reindex",
		mcplib.WithDescription("Re-index only files that changed since the last run."),
		mcplib.WithString("path",
			mcplib.Required(),
			mcplib.Description("Absolute path to the repository root."),
		),
	), handleReindex)

	s.AddTool(mcplib.NewTool("search",
		mcplib.WithDescription("Search for symbols by name, signature, or doc comment."),
		mcplib.WithString("query",
			mcplib.Required(),
			mcplib.Description("Search query."),
		),
	), handleSearch)

	s.AddTool(mcplib.NewTool("explore",
		mcplib.WithDescription("Get details about a symbol including its callers and callees."),
		mcplib.WithString("symbol",
			mcplib.Required(),
			mcplib.Description("Symbol name to explore."),
		),
	), handleExplore)

	s.AddTool(mcplib.NewTool("get_map",
		mcplib.WithDescription("Get a Mermaid diagram of the repo architecture."),
		mcplib.WithString("focus",
			mcplib.Description("Optional symbol name to focus the call graph."),
		),
	), handleGetMap)

	s.AddTool(mcplib.NewTool("list_repos",
		mcplib.WithDescription("List all indexed repositories."),
	), handleListRepos)
}

func jsonResult(v any) (*mcplib.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("mcp: marshal result: %w", err)
	}
	return mcplib.NewToolResultText(string(b)), nil
}

func handleIndexRepo(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	path := req.GetString("path", "")
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	store, dbPath, err := openStoreForRepo(path)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	_ = dbPath

	result, err := tools.IndexRepo(path, store)
	if err != nil {
		return nil, fmt.Errorf("mcp: index_repo: %w", err)
	}
	return jsonResult(map[string]any{
		"repo":            result.Repo,
		"path":            result.Path,
		"files_indexed":   result.FilesIndexed,
		"symbols_indexed": result.SymbolsIndexed,
		"duration_ms":     result.DurationMs,
	})
}

func handleReindex(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	path := req.GetString("path", "")
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	store, dbPath, err := openStoreForRepo(path)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	_ = dbPath

	result, err := tools.ReindexRepo(path, store)
	if err != nil {
		return nil, fmt.Errorf("mcp: reindex: %w", err)
	}
	return jsonResult(map[string]any{
		"repo":            result.Repo,
		"path":            result.Path,
		"files_indexed":   result.FilesIndexed,
		"symbols_indexed": result.SymbolsIndexed,
		"duration_ms":     result.DurationMs,
	})
}

func openStoreForRepo(repoPath string) (*storage.Store, string, error) {
	atlasDir, err := atlasDataDir()
	if err != nil {
		return nil, "", err
	}
	dbPath := filepath.Join(atlasDir, filepath.Base(repoPath)+".db")
	store, err := storage.Open(dbPath)
	if err != nil {
		return nil, "", fmt.Errorf("mcp: open store: %w", err)
	}
	return store, dbPath, nil
}

func handleSearch(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	atlasDir, err := atlasDataDir()
	if err != nil {
		return nil, err
	}
	result, err := search.Search(atlasDir, query)
	if err != nil {
		return nil, fmt.Errorf("mcp: search: %w", err)
	}
	return jsonResult(result)
}

func handleExplore(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	symbol := req.GetString("symbol", "")
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	atlasDir, err := atlasDataDir()
	if err != nil {
		return nil, err
	}
	result, err := tools.ExploreSymbol(atlasDir, symbol)
	if err != nil {
		return nil, fmt.Errorf("mcp: explore: %w", err)
	}
	if result == nil {
		return jsonResult(map[string]any{
			"symbol":  symbol,
			"kind":    "",
			"file":    "",
			"callers": []string{},
			"callees": []string{},
		})
	}
	return jsonResult(result)
}

func handleGetMap(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	focus := req.GetString("focus", "")
	atlasDir, err := atlasDataDir()
	if err != nil {
		return nil, err
	}
	result, err := tools.GetMap(atlasDir, focus)
	if err != nil {
		return nil, fmt.Errorf("mcp: get_map: %w", err)
	}
	return jsonResult(result)
}

func handleListRepos(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	atlasDir, err := atlasDataDir()
	if err != nil {
		return nil, err
	}
	repos, err := tools.ListRepos(atlasDir)
	if err != nil {
		return nil, fmt.Errorf("mcp: list_repos: %w", err)
	}
	if repos == nil {
		repos = []tools.RepoEntry{}
	}
	return jsonResult(map[string]any{
		"repos": repos,
	})
}

func atlasDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("mcp: get home dir: %w", err)
	}
	dir := filepath.Join(home, ".atlas")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mcp: create ~/.atlas: %w", err)
	}
	return dir, nil
}
