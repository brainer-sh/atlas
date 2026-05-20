//go:build with_embeddings && !linux

package embeddings

// preloadLibstdcxx is a no-op on non-Linux platforms.
func preloadLibstdcxx() {}
