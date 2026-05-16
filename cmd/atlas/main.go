package main

import (
	"log/slog"
	"os"

	"github.com/brainer.sh/atlas/internal/mcp"
)

func main() {
	s := mcp.New()
	if err := mcp.Serve(s); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}
