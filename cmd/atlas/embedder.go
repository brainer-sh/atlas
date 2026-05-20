//go:build !with_embeddings

package main

import "github.com/brainer.sh/atlas/internal/embeddings"

// newEmbedder returns nil when built without -tags with_embeddings.
// Search falls back to FTS-only.
func newEmbedder() embeddings.Embedder { return nil }
