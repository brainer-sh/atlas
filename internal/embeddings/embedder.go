// Package embeddings provides symbol embedding for semantic search.
package embeddings

import (
	"context"
	"errors"
	"math"
)

// ErrUnavailable is returned when the ONNX runtime or model files are not present.
// Build with -tags with_embeddings and run scripts/download-deps.sh first.
var ErrUnavailable = errors.New("embeddings: ONNX runtime not available (build with -tags with_embeddings)")

// Embedder converts text into a fixed-size float32 vector.
type Embedder interface {
	// Embed returns one unit-normalized vector per input text.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// Dim returns the vector dimension.
	Dim() int
	Close() error
}

// FakeEmbedder returns deterministic unit vectors. Used in tests only.
// Vector dimension is 4.
type FakeEmbedder struct{}

const fakeDim = 4

// Embed returns a deterministic unit vector for each input text.
func (FakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vecs := make([][]float32, len(texts))
	for i, t := range texts {
		vecs[i] = fakeVec(t)
	}
	return vecs, nil
}

// Dim returns the dimension of FakeEmbedder vectors.
func (FakeEmbedder) Dim() int { return fakeDim }

// Close is a no-op.
func (FakeEmbedder) Close() error { return nil }

// fakeVec produces a deterministic unit vector from a string.
func fakeVec(s string) []float32 {
	v := make([]float32, fakeDim)
	for i, c := range s {
		v[i%fakeDim] += float32(c)
	}
	var norm float32
	for _, x := range v {
		norm += x * x
	}
	if norm > 0 {
		norm = float32(math.Sqrt(float64(norm)))
		for i := range v {
			v[i] /= norm
		}
	}
	return v
}
