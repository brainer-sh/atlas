//go:build with_embeddings

package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode"

	ort "github.com/yalue/onnxruntime_go"
)

const (
	onnxDim    = 384
	onnxMaxLen = 128
)

var (
	ortOnce sync.Once
	ortErr  error
)

func initORT(libPath string) error {
	ortOnce.Do(func() {
		ort.SetSharedLibraryPath(libPath)
		ortErr = ort.InitializeEnvironment()
	})
	return ortErr
}

// OnnxEmbedder runs the all-MiniLM-L6-v2 ONNX model for semantic embedding.
// Vectors are 384-dimensional and unit-normalized.
type OnnxEmbedder struct {
	session *ort.DynamicAdvancedSession
	tok     *wordpieceTokenizer
}

// NewOnnxEmbedder loads the ONNX model and prepares the inference session.
// Requires scripts/download-deps.sh to have been run first.
func NewOnnxEmbedder() (Embedder, error) {
	libPath := atlasLibPath()
	modelPath := atlasModelPath()
	tokPath := atlasTokPath()

	for _, p := range []string{libPath, modelPath, tokPath} {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("embeddings: %s missing - run scripts/download-deps.sh: %w", p, err)
		}
	}

	if err := initORT(libPath); err != nil {
		return nil, fmt.Errorf("embeddings: init onnxruntime: %w", err)
	}

	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("embeddings: create session: %w", err)
	}

	tok, err := loadTokenizer(tokPath)
	if err != nil {
		session.Destroy()
		return nil, fmt.Errorf("embeddings: load tokenizer: %w", err)
	}

	return &OnnxEmbedder{session: session, tok: tok}, nil
}

// Dim returns the embedding dimension.
func (e *OnnxEmbedder) Dim() int { return onnxDim }

// Close destroys the ONNX session.
func (e *OnnxEmbedder) Close() error {
	return e.session.Destroy()
}

// Embed returns one unit-normalized 384-dim vector per input text.
func (e *OnnxEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vecs := make([][]float32, len(texts))
	for i, text := range texts {
		v, err := e.embedOne(text)
		if err != nil {
			return nil, fmt.Errorf("embeddings: embed %q: %w", text, err)
		}
		vecs[i] = v
	}
	return vecs, nil
}

func (e *OnnxEmbedder) embedOne(text string) ([]float32, error) {
	ids, mask, typeIDs := e.tok.encode(text, onnxMaxLen)
	seqLen := int64(len(ids))
	shape := ort.NewShape(1, seqLen)

	inputIDsTensor, err := ort.NewTensor(shape, ids)
	if err != nil {
		return nil, fmt.Errorf("input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	attMaskTensor, err := ort.NewTensor(shape, mask)
	if err != nil {
		return nil, fmt.Errorf("attention_mask tensor: %w", err)
	}
	defer attMaskTensor.Destroy()

	typeIDsTensor, err := ort.NewTensor(shape, typeIDs)
	if err != nil {
		return nil, fmt.Errorf("token_type_ids tensor: %w", err)
	}
	defer typeIDsTensor.Destroy()

	outTensor, err := ort.NewEmptyTensor[float32](ort.NewShape(1, seqLen, onnxDim))
	if err != nil {
		return nil, fmt.Errorf("output tensor: %w", err)
	}
	defer outTensor.Destroy()

	if err := e.session.Run(
		[]ort.Value{inputIDsTensor, attMaskTensor, typeIDsTensor},
		[]ort.Value{outTensor},
	); err != nil {
		return nil, fmt.Errorf("onnx run: %w", err)
	}

	hidden := outTensor.GetData()
	vec := meanPool(hidden, mask, int(seqLen), onnxDim)
	return l2Normalize(vec), nil
}

// meanPool computes the attention-masked mean over the sequence dimension.
func meanPool(hidden []float32, mask []int64, seqLen, dim int) []float32 {
	out := make([]float32, dim)
	var total float64
	for t := 0; t < seqLen; t++ {
		if mask[t] == 0 {
			continue
		}
		total++
		for d := 0; d < dim; d++ {
			out[d] += hidden[t*dim+d]
		}
	}
	if total > 0 {
		for d := range out {
			out[d] = float32(float64(out[d]) / total)
		}
	}
	return out
}

// l2Normalize normalizes a vector to unit length in-place.
func l2Normalize(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	norm := math.Sqrt(sum)
	if norm < 1e-12 {
		return v
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / norm)
	}
	return v
}

func atlasLibPath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, ".atlas", "lib", "libonnxruntime.dylib")
	case "windows":
		return filepath.Join(home, ".atlas", "lib", "onnxruntime.dll")
	default:
		return filepath.Join(home, ".atlas", "lib", "libonnxruntime.so")
	}
}

func atlasModelPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".atlas", "models", "all-MiniLM-L6-v2.onnx")
}

func atlasTokPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".atlas", "models", "tokenizer.json")
}

// --- WordPiece tokenizer ---

type wordpieceTokenizer struct {
	vocab     map[string]int64
	unkID     int64
	clsID     int64
	sepID     int64
	padID     int64
	lowercase bool
}

type tokenizerJSON struct {
	Model struct {
		Vocab    map[string]int64 `json:"vocab"`
		UnkToken string           `json:"unk_token"`
	} `json:"model"`
	Normalizer *struct {
		Lowercase bool `json:"lowercase"`
	} `json:"normalizer"`
}

func loadTokenizer(path string) (*wordpieceTokenizer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var tj tokenizerJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if len(tj.Model.Vocab) == 0 {
		return nil, fmt.Errorf("tokenizer: model.vocab is empty")
	}
	get := func(tok string) int64 {
		id, ok := tj.Model.Vocab[tok]
		if !ok {
			return tj.Model.Vocab[tj.Model.UnkToken]
		}
		return id
	}
	lowercase := true
	if tj.Normalizer != nil {
		lowercase = tj.Normalizer.Lowercase
	}
	return &wordpieceTokenizer{
		vocab:     tj.Model.Vocab,
		unkID:     get("[UNK]"),
		clsID:     get("[CLS]"),
		sepID:     get("[SEP]"),
		padID:     get("[PAD]"),
		lowercase: lowercase,
	}, nil
}

// encode tokenizes text and returns (input_ids, attention_mask, token_type_ids).
// Truncated to maxLen tokens including [CLS] and [SEP].
func (t *wordpieceTokenizer) encode(text string, maxLen int) (ids, mask, typeIDs []int64) {
	if t.lowercase {
		text = strings.ToLower(text)
	}
	words := bertPreTokenize(text)

	tokens := []int64{t.clsID}
	for _, word := range words {
		for _, id := range t.wordpiece(word) {
			tokens = append(tokens, id)
			if len(tokens) >= maxLen-1 {
				goto done
			}
		}
	}
done:
	tokens = append(tokens, t.sepID)

	n := len(tokens)
	ids = make([]int64, n)
	mask = make([]int64, n)
	typeIDs = make([]int64, n)
	copy(ids, tokens)
	for i := range ids {
		mask[i] = 1
	}
	return
}

// wordpiece applies WordPiece segmentation to a single word.
func (t *wordpieceTokenizer) wordpiece(word string) []int64 {
	if id, ok := t.vocab[word]; ok {
		return []int64{id}
	}
	runes := []rune(word)
	var out []int64
	start := 0
	for start < len(runes) {
		end := len(runes)
		found := false
		for end > start {
			sub := string(runes[start:end])
			if start > 0 {
				sub = "##" + sub
			}
			if id, ok := t.vocab[sub]; ok {
				out = append(out, id)
				start = end
				found = true
				break
			}
			end--
		}
		if !found {
			return []int64{t.unkID}
		}
	}
	return out
}

// bertPreTokenize splits text on whitespace and punctuation.
func bertPreTokenize(text string) []string {
	var words []string
	var cur strings.Builder
	for _, r := range text {
		if unicode.IsSpace(r) {
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
		} else if unicode.IsPunct(r) {
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
			words = append(words, string(r))
		} else {
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}
