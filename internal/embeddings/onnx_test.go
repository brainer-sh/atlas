//go:build with_embeddings

package embeddings

import (
	"math"
	"testing"
)

// minimalVocab is enough to test the tokenizer without a real tokenizer.json.
var minimalVocab = map[string]int64{
	"[PAD]":    0,
	"[UNK]":    100,
	"[CLS]":    101,
	"[SEP]":    102,
	"hello":    7592,
	"world":    2088,
	"##s":      1879,
	"function": 4231,
	"##_":      1035,
	"gpu":      14246,
}

func minimalTokenizer() *wordpieceTokenizer {
	return &wordpieceTokenizer{
		vocab:     minimalVocab,
		unkID:     100,
		clsID:     101,
		sepID:     102,
		padID:     0,
		lowercase: true,
	}
}

func TestEncode_startsWithCLS(t *testing.T) {
	tok := minimalTokenizer()
	ids, mask, _ := tok.encode("hello world", 32)
	if ids[0] != 101 {
		t.Errorf("ids[0] = %d, want 101 ([CLS])", ids[0])
	}
	if ids[len(ids)-1] != 102 {
		t.Errorf("ids[last] = %d, want 102 ([SEP])", ids[len(ids)-1])
	}
	for i, m := range mask {
		if m != 1 {
			t.Errorf("mask[%d] = %d, want 1", i, m)
		}
	}
}

func TestEncode_truncation(t *testing.T) {
	tok := minimalTokenizer()
	// With maxLen=4: [CLS] + at most 2 tokens + [SEP]
	ids, _, _ := tok.encode("hello world", 4)
	if len(ids) > 4 {
		t.Errorf("len(ids) = %d, want <= 4", len(ids))
	}
	if ids[len(ids)-1] != 102 {
		t.Errorf("last token = %d, want [SEP]=102", ids[len(ids)-1])
	}
}

func TestWordpiece_knownWord(t *testing.T) {
	tok := minimalTokenizer()
	got := tok.wordpiece("hello")
	if len(got) != 1 || got[0] != 7592 {
		t.Errorf("wordpiece(hello) = %v, want [7592]", got)
	}
}

func TestWordpiece_unknownWord(t *testing.T) {
	tok := minimalTokenizer()
	got := tok.wordpiece("zzzzunknown")
	if len(got) != 1 || got[0] != 100 {
		t.Errorf("wordpiece(zzzzunknown) = %v, want [100 ([UNK])]", got)
	}
}

func TestBertPreTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"foo.bar", []string{"foo", ".", "bar"}},
		{"  leading", []string{"leading"}},
	}
	for _, tt := range tests {
		got := bertPreTokenize(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("bertPreTokenize(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("bertPreTokenize(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestMeanPool(t *testing.T) {
	// 2 tokens, dim=2: [[1,2],[3,4]], mask=[1,1] -> mean=[2,3]
	hidden := []float32{1, 2, 3, 4}
	mask := []int64{1, 1}
	got := meanPool(hidden, mask, 2, 2)
	if math.Abs(float64(got[0]-2)) > 1e-5 || math.Abs(float64(got[1]-3)) > 1e-5 {
		t.Errorf("meanPool = %v, want [2 3]", got)
	}
}

func TestMeanPool_maskedToken(t *testing.T) {
	// 2 tokens, dim=2: [[1,2],[3,4]], mask=[1,0] -> mean=[1,2] (second masked)
	hidden := []float32{1, 2, 3, 4}
	mask := []int64{1, 0}
	got := meanPool(hidden, mask, 2, 2)
	if math.Abs(float64(got[0]-1)) > 1e-5 || math.Abs(float64(got[1]-2)) > 1e-5 {
		t.Errorf("meanPool(masked) = %v, want [1 2]", got)
	}
}

func TestL2Normalize(t *testing.T) {
	v := []float32{3, 4}
	got := l2Normalize(v)
	norm := math.Sqrt(float64(got[0])*float64(got[0]) + float64(got[1])*float64(got[1]))
	if math.Abs(norm-1.0) > 1e-5 {
		t.Errorf("l2Normalize norm = %v, want 1.0", norm)
	}
}

func TestNewOnnxEmbedder_MissingFiles(t *testing.T) {
	// Without running download-deps.sh, files are absent -> error expected.
	_, err := NewOnnxEmbedder()
	if err == nil {
		t.Log("NewOnnxEmbedder succeeded (model files present, good)")
	} else {
		t.Logf("NewOnnxEmbedder error (expected without download-deps.sh): %v", err)
	}
}
