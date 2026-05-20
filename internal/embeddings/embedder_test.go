package embeddings

import (
	"context"
	"math"
	"testing"
)

func TestFakeEmbedder_Embed(t *testing.T) {
	e := FakeEmbedder{}
	if e.Dim() != fakeDim {
		t.Fatalf("Dim() = %d, want %d", e.Dim(), fakeDim)
	}

	texts := []string{"hello", "world", "hello"}
	vecs, err := e.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("Embed() error: %v", err)
	}
	if len(vecs) != len(texts) {
		t.Fatalf("len(vecs) = %d, want %d", len(vecs), len(texts))
	}

	// Each vector must be unit length.
	for i, v := range vecs {
		var norm float64
		for _, x := range v {
			norm += float64(x) * float64(x)
		}
		norm = math.Sqrt(norm)
		if math.Abs(norm-1.0) > 1e-5 {
			t.Errorf("vecs[%d] norm = %v, want 1.0", i, norm)
		}
	}

	// Same input must give same output.
	for i := range vecs[0] {
		if vecs[0][i] != vecs[2][i] {
			t.Errorf("vecs[0][%d] = %v, vecs[2][%d] = %v: same input should give same vector",
				i, vecs[0][i], i, vecs[2][i])
		}
	}
}
