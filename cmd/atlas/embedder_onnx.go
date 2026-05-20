//go:build with_embeddings

package main

import (
	"log/slog"

	"github.com/brainer.sh/atlas/internal/embeddings"
)

// newEmbedder returns an OnnxEmbedder when built with -tags with_embeddings.
// Falls back to nil (FTS-only) if the model files are missing.
func newEmbedder() embeddings.Embedder {
	e, err := embeddings.NewOnnxEmbedder()
	if err != nil {
		slog.Warn("embeddings unavailable, falling back to FTS", "err", err)
		return nil
	}
	return e
}
