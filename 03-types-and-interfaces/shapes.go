// Package interfaces covers structs, methods, embedding and interfaces: Go's alternative to classes and inheritance.
package interfaces

import (
	"fmt"
	"math"
	"strings"
)

// Shape is an INTERFACE: a set of method signatures. Any type with these methods satisfies it
// automatically (no "implements" keyword). Keep interfaces small; define them where they are USED.
type Shape interface {
	Area() float64
	Perimeter() float64
}

// Rect is a STRUCT: a collection of named fields.
type Rect struct {
	Width, Height float64
}

// Area has a VALUE receiver: it works on a copy and can't modify the original.
func (r Rect) Area() float64      { return r.Width * r.Height }
func (r Rect) Perimeter() float64 { return 2 * (r.Width + r.Height) }

// Scale has a POINTER receiver: it modifies the original. Rule of thumb: if any method needs a pointer
// receiver (or the struct is large), use pointer receivers for all methods of that type.
func (r *Rect) Scale(factor float64) {
	r.Width *= factor
	r.Height *= factor
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }

// String makes Circle a fmt.Stringer: fmt.Println uses it automatically.
func (c Circle) String() string { return fmt.Sprintf("Circle(r=%.1f)", c.Radius) }

// Compile-time check that the types satisfy the interface (a common idiom)
var (
	_ Shape = Rect{}
	_ Shape = Circle{}
)

// TotalArea works with ANY Shape: polymorphism through interfaces.
func TotalArea(shapes ...Shape) float64 {
	total := 0.0
	for _, s := range shapes {
		total += s.Area()
	}
	return total
}

// Describe uses a TYPE SWITCH to act on the concrete type behind an interface (or `any`).
func Describe(v any) string {
	switch x := v.(type) {
	case Rect:
		return fmt.Sprintf("rectangle %vx%v", x.Width, x.Height)
	case fmt.Stringer:
		return "stringer " + x.String()
	case int, float64:
		return fmt.Sprintf("number %v", x)
	case nil:
		return "nothing"
	default:
		return fmt.Sprintf("unknown %T", x)
	}
}

// ---- Composition with EMBEDDING (instead of inheritance) ----

type Logger struct {
	Prefix string
	lines  []string
}

func (l *Logger) Log(msg string) { l.lines = append(l.lines, l.Prefix+msg) }
func (l *Logger) Lines() string  { return strings.Join(l.lines, "; ") }

// OrderService EMBEDS *Logger: Logger's fields and methods are PROMOTED, so svc.Log(...) works.
// It's composition: OrderService HAS a Logger, it is not a subtype of Logger.
type OrderService struct {
	*Logger
	orders int
}

func NewOrderService() *OrderService { // constructor functions are named NewXxx by convention
	return &OrderService{Logger: &Logger{Prefix: "[orders] "}}
}

func (s *OrderService) Place(item string) {
	s.orders++
	s.Log("placed " + item)
}
