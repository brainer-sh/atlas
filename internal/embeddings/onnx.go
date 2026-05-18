//go:build with_embeddings

package embeddings

// This file requires:
//   - scripts/download-deps.sh to have been run (downloads model + onnxruntime lib)
//   - yalue/onnxruntime_go (add to go.mod when implementing)
//
// Build: go build -tags with_embeddings ./...

import (
	"context"
	"fmt"
)

// OnnxEmbedder runs the all-MiniLM-L6-v2 ONNX model for semantic embedding.
// Vectors are 384-dimensional and unit-normalized.
type OnnxEmbedder struct {
	dim int
}

// NewOnnxEmbedder loads the ONNX model and prepares the inference session.
func NewOnnxEmbedder() (Embedder, error) {
	// TODO: implement with yalue/onnxruntime_go
	//   1. Extract native lib to ~/.atlas/lib/ on first run
	//   2. Load model from internal/embeddings/model/all-MiniLM-L6-v2.onnx
	//   3. Load tokenizer from internal/embeddings/model/tokenizer.json
	//   4. Create OrtSession
	return nil, fmt.Errorf("embeddings: OnnxEmbedder not yet implemented")
}

func (e *OnnxEmbedder) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, fmt.Errorf("embeddings: not implemented")
}

func (e *OnnxEmbedder) Dim() int   { return e.dim }
func (e *OnnxEmbedder) Close() error { return nil }
