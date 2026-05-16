package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
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
	return jsonResult(map[string]any{
		"repo":            "",
		"path":            req.GetString("path", ""),
		"files_indexed":   0,
		"symbols_indexed": 0,
		"duration_ms":     0,
	})
}

func handleReindex(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return jsonResult(map[string]any{
		"repo":            "",
		"path":            req.GetString("path", ""),
		"files_indexed":   0,
		"symbols_indexed": 0,
		"duration_ms":     0,
	})
}

func handleSearch(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return jsonResult(map[string]any{
		"query":   req.GetString("query", ""),
		"results": []any{},
	})
}

func handleExplore(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return jsonResult(map[string]any{
		"symbol":  req.GetString("symbol", ""),
		"kind":    "",
		"file":    "",
		"callers": []string{},
		"callees": []string{},
	})
}

func handleGetMap(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return jsonResult(map[string]any{
		"focus":   nil,
		"diagram": "",
	})
}

func handleListRepos(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return jsonResult(map[string]any{
		"repos": []any{},
	})
}
