package interfaces

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func Example_structs() {
	r := Rect{Width: 2, Height: 3} // composite literal with field names
	p := &r                        // pointer to the struct; p.Width works (auto-dereference)
	p.Scale(2)
	fmt.Println(r, r.Area())

	type point struct{ X, Y int } // anonymous/local types are fine for small helpers
	fmt.Printf("%+v\n", point{1, 2})
	// Output:
	// {4 6} 24
	// {X:1 Y:2}
}

func ExampleTotalArea() {
	fmt.Printf("%.2f\n", TotalArea(Rect{Width: 2, Height: 3}, Circle{Radius: 1}))
	// Output: 9.14
}

func ExampleDescribe() {
	fmt.Println(Describe(Rect{Width: 1, Height: 2}))
	fmt.Println(Describe(Circle{Radius: 3}))
	fmt.Println(Describe(42))
	fmt.Println(Describe(nil))
	fmt.Println(Describe([]int{1}))
	// Output:
	// rectangle 1x2
	// stringer Circle(r=3.0)
	// number 42
	// nothing
	// unknown []int
}

func Example_typeAssertion() {
	var s Shape = Circle{Radius: 2}
	if c, ok := s.(Circle); ok { // "comma ok" form never panics
		fmt.Println("it's a circle of radius", c.Radius)
	}
	_, isRect := s.(Rect)
	fmt.Println(isRect)
	// Output:
	// it's a circle of radius 2
	// false
}

func ExampleNewOrderService() {
	svc := NewOrderService()
	svc.Place("book")
	svc.Place("pen")
	fmt.Println(svc.orders, "|", svc.Lines()) // Lines() is promoted from the embedded *Logger
	// Output: 2 | [orders] placed book; [orders] placed pen
}

// The most important interfaces in Go are tiny ones from the standard library, e.g.
// io.Reader { Read(p []byte) (int, error) } and io.Writer { Write(p []byte) (int, error) }.
// Files, network connections, HTTP bodies, buffers and gzip streams all implement them, so they compose.
func Example_ioInterfaces() {
	var r io.Reader = strings.NewReader("hello, interfaces\n")
	var w io.Writer = os.Stdout
	io.Copy(w, r)
	// Output: hello, interfaces
}
