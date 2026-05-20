package mcp

import (
	"log/slog"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/brainer.sh/atlas/internal/embeddings"
)

// New creates and configures the Atlas MCP server.
// e may be nil; in that case search falls back to FTS-only.
func New(e embeddings.Embedder) *mcpserver.MCPServer {
	s := mcpserver.NewMCPServer("atlas", "0.1.0")
	registerTools(s, e)
	slog.Info("mcp server initialized")
	return s
}

// Serve starts the MCP server on stdio and blocks until the client disconnects.
func Serve(s *mcpserver.MCPServer) error {
	return mcpserver.ServeStdio(s)
}
