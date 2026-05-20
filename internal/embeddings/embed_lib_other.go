//go:build with_embeddings && !((linux && amd64) || (linux && arm64) || (darwin && arm64) || (darwin && amd64))

package embeddings

// nativeLib is nil on unsupported platforms; NewOnnxEmbedder returns an error.
var nativeLib []byte
