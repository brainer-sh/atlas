package simple

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

// Run demonstrates Add.
func Run() int {
	return Add(1, 2)
}
