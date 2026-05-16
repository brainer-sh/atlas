package goindexer

import (
	"testing"
)

var fixtureSource = []byte(`package simple

import "fmt"

// Greeter greets people.
type Greeter struct {
	Name string
}

// Greet returns a greeting.
func (g *Greeter) Greet() string {
	return fmt.Sprintf("Hello, %s!", g.Name)
}

// Sayer can say things.
type Sayer interface {
	Say() string
}

// Add adds two integers.
func Add(a, b int) int {
	return a + b
}
`)

func TestNew(t *testing.T) {
	idx, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer idx.Close()
}

func TestIndexSource_Package(t *testing.T) {
	idx, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer idx.Close()

	fi, err := idx.IndexSource(fixtureSource)
	if err != nil {
		t.Fatalf("IndexSource() error: %v", err)
	}
	if fi.Package != "simple" {
		t.Errorf("Package = %q, want %q", fi.Package, "simple")
	}
}

func TestIndexSource_Imports(t *testing.T) {
	idx, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer idx.Close()

	fi, err := idx.IndexSource(fixtureSource)
	if err != nil {
		t.Fatalf("IndexSource() error: %v", err)
	}
	if len(fi.Imports) != 1 || fi.Imports[0] != "fmt" {
		t.Errorf("Imports = %v, want [fmt]", fi.Imports)
	}
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
		doc  string
	}{
		{name: "Greeter", kind: "struct", doc: "Greeter greets people."},
		{name: "Greet", kind: "method", doc: "Greet returns a greeting."},
		{name: "Sayer", kind: "interface", doc: "Sayer can say things."},
		{name: "Add", kind: "function", doc: "Add adds two integers."},
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
			if s.Doc != tt.doc {
				t.Errorf("Doc = %q, want %q", s.Doc, tt.doc)
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

	for _, s := range fi.Symbols {
		if s.Name == "Add" {
			if s.Signature == "" {
				t.Error("Add signature is empty")
			}
			return
		}
	}
	t.Error("Add symbol not found")
}
