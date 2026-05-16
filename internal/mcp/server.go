package mcp

import (
	"log/slog"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// New creates and configures the Atlas MCP server with all tool stubs registered.
func New() *mcpserver.MCPServer {
	s := mcpserver.NewMCPServer("atlas", "0.1.0")
	registerTools(s)
	slog.Info("mcp server initialized")
	return s
}

// Serve starts the MCP server on stdio and blocks until the client disconnects.
func Serve(s *mcpserver.MCPServer) error {
	return mcpserver.ServeStdio(s)
}
