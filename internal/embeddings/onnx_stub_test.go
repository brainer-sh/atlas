//go:build !with_embeddings

package embeddings

import "testing"

func TestNewOnnxEmbedder_Stub(t *testing.T) {
	_, err := NewOnnxEmbedder()
	if err == nil {
		t.Fatal("expected error from stub, got nil")
	}
}
