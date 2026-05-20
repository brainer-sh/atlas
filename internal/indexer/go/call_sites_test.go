package goindexer

import (
	"testing"
)

var callSiteFixture = []byte(`package simple

func Add(a, b int) int {
	return a + b
}

func Run() int {
	return Add(1, 2)
}

func (g *Greeter) Greet() string {
	return fmt.Sprintf("Hello, %s!", g.Name)
}
`)

func TestExtractCallSites_basic(t *testing.T) {
	idx, err := New()
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	defer idx.Close()

	fi, err := idx.IndexSource(callSiteFixture)
	if err != nil {
		t.Fatalf("IndexSource(): %v", err)
	}

	byCallee := make(map[string]CallSite)
	for _, cs := range fi.CallSites {
		byCallee[cs.CalleeName] = cs
	}

	tests := []struct {
		callee  string
		wantMin uint
		wantMax uint
	}{
		{callee: "Add", wantMin: 8, wantMax: 8},
		{callee: "Sprintf", wantMin: 12, wantMax: 12},
	}
	for _, tt := range tests {
		t.Run(tt.callee, func(t *testing.T) {
			cs, ok := byCallee[tt.callee]
			if !ok {
				t.Fatalf("call site %q not found; all: %v", tt.callee, fi.CallSites)
			}
			if cs.Line < tt.wantMin || cs.Line > tt.wantMax {
				t.Errorf("Line = %d, want %d-%d", cs.Line, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestExtractCallSites_none(t *testing.T) {
	idx, err := New()
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	defer idx.Close()

	src := []byte(`package simple

func Pure(a, b int) int {
	return a + b
}
`)
	fi, err := idx.IndexSource(src)
	if err != nil {
		t.Fatalf("IndexSource(): %v", err)
	}
	if len(fi.CallSites) != 0 {
		t.Errorf("CallSites = %v, want none", fi.CallSites)
	}
}
