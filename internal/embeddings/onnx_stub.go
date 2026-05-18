//go:build !with_embeddings

package embeddings

// NewOnnxEmbedder returns ErrUnavailable when built without the with_embeddings tag.
// To enable: run scripts/download-deps.sh then build with -tags with_embeddings.
func NewOnnxEmbedder() (Embedder, error) {
	return nil, ErrUnavailable
}
