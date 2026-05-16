package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func fixtureDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "go", "simple")
}

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
}

func TestHandlers(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		handler func(context.Context, mcplib.CallToolRequest) (*mcplib.CallToolResult, error)
		args    map[string]any
		wantKey string
	}{
		{
			name:    "index_repo",
			handler: handleIndexRepo,
			args:    map[string]any{"path": fixtureDir()},
			wantKey: "files_indexed",
		},
		{
			name:    "reindex",
			handler: handleReindex,
			args:    map[string]any{"path": fixtureDir()},
			wantKey: "files_indexed",
		},
		{
			name:    "search",
			handler: handleSearch,
			args:    map[string]any{"query": "foo"},
			wantKey: "results",
		},
		{
			name:    "explore",
			handler: handleExplore,
			args:    map[string]any{"symbol": "Foo"},
			wantKey: "callers",
		},
		{
			name:    "get_map",
			handler: handleGetMap,
			args:    map[string]any{},
			wantKey: "diagram",
		},
		{
			name:    "list_repos",
			handler: handleListRepos,
			args:    map[string]any{},
			wantKey: "repos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcplib.CallToolRequest{}
			req.Params.Arguments = map[string]any(tt.args)

			result, err := tt.handler(ctx, req)
			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}
			if result == nil {
				t.Fatal("handler returned nil result")
			}
			if len(result.Content) == 0 {
				t.Fatal("handler returned empty content")
			}

			text, ok := result.Content[0].(mcplib.TextContent)
			if !ok {
				t.Fatal("expected TextContent")
			}

			var got map[string]any
			if err := json.Unmarshal([]byte(text.Text), &got); err != nil {
				t.Fatalf("result is not valid JSON: %v", err)
			}
			if _, ok := got[tt.wantKey]; !ok {
				t.Errorf("response missing key %q", tt.wantKey)
			}
		})
	}
}
