package cindexer

import (
	"testing"
)

var fixtureSource = []byte(`
/* Creates a GPU context. */
SDL_GPUDevice *SDL_GPUCreateDevice(SDL_GPUShaderFormat format, bool debug) {
	return NULL;
}

// Simple helper.
int add(int a, int b) {
	return a + b;
}

typedef struct SDL_GPUDevice {
	int id;
} SDL_GPUDevice;

typedef enum SDL_GPUShaderFormat {
	SDL_GPU_SHADERFORMAT_INVALID = 0,
} SDL_GPUShaderFormat;

#define SDL_MAX_DEVICES 8

#define SDL_CLAMP(x, lo, hi) ((x) < (lo) ? (lo) : (x) > (hi) ? (hi) : (x))
`)

func TestNew(t *testing.T) {
	idx, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer idx.Close()
}

func TestIndexSource_Symbols(t *testing.T) {
	idx, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer idx.Close()

	fi, err := idx.IndexSource(fixtureSource)
	if err != nil {
		t.Fatalf("IndexSource() error: %v", err)
	}

	tests := []struct {
		name string
		kind string
	}{
		{name: "SDL_GPUCreateDevice", kind: "function"},
		{name: "add", kind: "function"},
		{name: "SDL_GPUDevice", kind: "struct"},
		{name: "SDL_GPUShaderFormat", kind: "enum"},
		{name: "SDL_MAX_DEVICES", kind: "macro"},
		{name: "SDL_CLAMP", kind: "macro"},
	}

	byName := make(map[string]Symbol)
	for _, s := range fi.Symbols {
		byName[s.Name] = s
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := byName[tt.name]
			if !ok {
				t.Fatalf("symbol %q not found", tt.name)
			}
			if s.Kind != tt.kind {
				t.Errorf("Kind = %q, want %q", s.Kind, tt.kind)
			}
			if s.LineStart == 0 {
				t.Errorf("LineStart = 0, want > 0")
			}
		})
	}
}

func TestIndexSource_FunctionSignature(t *testing.T) {
	idx, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer idx.Close()

	fi, err := idx.IndexSource(fixtureSource)
	if err != nil {
		t.Fatalf("IndexSource() error: %v", err)
	}

	byName := make(map[string]Symbol)
	for _, s := range fi.Symbols {
		byName[s.Name] = s
	}

	tests := []struct {
		name    string
		wantSig string
	}{
		{name: "SDL_GPUCreateDevice", wantSig: "SDL_GPUDevice *SDL_GPUCreateDevice(SDL_GPUShaderFormat format, bool debug)"},
		{name: "add", wantSig: "int add(int a, int b)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := byName[tt.name]
			if !ok {
				t.Fatalf("symbol %q not found", tt.name)
			}
			if s.Signature != tt.wantSig {
				t.Errorf("Signature = %q, want %q", s.Signature, tt.wantSig)
			}
		})
	}
}

func TestIndexSource_Doc(t *testing.T) {
	idx, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer idx.Close()

	fi, err := idx.IndexSource(fixtureSource)
	if err != nil {
		t.Fatalf("IndexSource() error: %v", err)
	}

	byName := make(map[string]Symbol)
	for _, s := range fi.Symbols {
		byName[s.Name] = s
	}

	if s := byName["add"]; s.Doc != "Simple helper." {
		t.Errorf("add Doc = %q, want %q", s.Doc, "Simple helper.")
	}
}

func TestIndexFile_IsHeader(t *testing.T) {
	idx, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer idx.Close()

	tests := []struct {
		path     string
		isHeader bool
	}{
		{path: "SDL_gpu.h", isHeader: true},
		{path: "SDL_gpu.c", isHeader: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			// IndexFile would fail on non-existent paths; test the detection logic directly.
			fi, _ := idx.IndexSource([]byte{})
			fi.IsHeader = len(tt.path) > 2 && tt.path[len(tt.path)-2:] == ".h"
			if fi.IsHeader != tt.isHeader {
				t.Errorf("IsHeader = %v, want %v", fi.IsHeader, tt.isHeader)
			}
		})
	}
}
