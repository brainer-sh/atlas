package cppindexer

import (
	"testing"
)

var fixtureSource = []byte(`
// Creates a renderer.
Renderer *CreateRenderer(int width, int height) {
	return nullptr;
}

class Shape {
public:
	virtual float area() const = 0;
	void setColor(int r, int g, int b);
};

void Shape::setColor(int r, int g, int b) {
	// implementation
}

template<typename T>
class Stack {
public:
	void push(T val);
	T pop();
};

template<typename T>
void process(T value) {
	// implementation
}

struct Point {
	float x;
	float y;
};

enum class Direction {
	North,
	South,
};

#define MAX_ITEMS 64
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
		{name: "CreateRenderer", kind: "function"},
		{name: "Shape", kind: "class"},
		{name: "Shape::setColor", kind: "method"},
		{name: "Stack", kind: "template"},
		{name: "process", kind: "template"},
		{name: "Point", kind: "struct"},
		{name: "Direction", kind: "enum"},
		{name: "MAX_ITEMS", kind: "macro"},
	}

	byName := make(map[string]Symbol)
	for _, s := range fi.Symbols {
		byName[s.Name] = s
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := byName[tt.name]
			if !ok {
				t.Fatalf("symbol %q not found (got: %v)", tt.name, symbolNames(fi.Symbols))
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
		{name: "CreateRenderer", wantSig: "Renderer *CreateRenderer(int width, int height)"},
		{name: "Shape::setColor", wantSig: "void Shape::setColor(int r, int g, int b)"},
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

	if s := byName["CreateRenderer"]; s.Doc != "Creates a renderer." {
		t.Errorf("CreateRenderer Doc = %q, want %q", s.Doc, "Creates a renderer.")
	}
}

func TestIndexFile_IsHeader(t *testing.T) {
	tests := []struct {
		path     string
		isHeader bool
	}{
		{path: "renderer.hpp", isHeader: true},
		{path: "renderer.hxx", isHeader: true},
		{path: "renderer.hh", isHeader: true},
		{path: "renderer.h", isHeader: true},
		{path: "renderer.cpp", isHeader: false},
		{path: "renderer.cc", isHeader: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isCppHeader(tt.path)
			if got != tt.isHeader {
				t.Errorf("isCppHeader(%q) = %v, want %v", tt.path, got, tt.isHeader)
			}
		})
	}
}

func symbolNames(symbols []Symbol) []string {
	names := make([]string, len(symbols))
	for i, s := range symbols {
		names[i] = s.Name + "(" + s.Kind + ")"
	}
	return names
}
