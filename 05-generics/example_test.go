package generics

import (
	"fmt"
	"strconv"
	"strings"
)

type Product struct {
	Name  string
	Price float64
}

var products = []Product{{"pen", 2.5}, {"book", 15}, {"lamp", 40}}

func ExampleMap() {
	fmt.Println(Map(products, func(p Product) string { return strings.ToUpper(p.Name) }))
	fmt.Println(Map([]int{1, 2, 3}, strconv.Itoa)) // T=int, R=string, inferred
	// Output:
	// [PEN BOOK LAMP]
	// [1 2 3]
}

func ExampleFilter() {
	cheap := Filter(products, func(p Product) bool { return p.Price < 20 })
	fmt.Println(len(cheap), cheap[1].Name)
	// Output: 2 book
}

func ExampleReduce() {
	total := Reduce(products, 0.0, func(sum float64, p Product) float64 { return sum + p.Price })
	fmt.Println(total)
	// Output: 57.5
}

type Rupees int // a named type whose underlying type is int: allowed by ~int

func ExampleSum() {
	fmt.Println(Sum(1, 2, 3), Sum(1.5, 2.5), Sum[Rupees](100, 250))
	// Output: 6 4 350
}

func ExampleMaxBy() {
	p, ok := MaxBy(products, func(p Product) float64 { return p.Price })
	fmt.Println(p.Name, ok)
	_, ok = MaxBy([]Product{}, func(p Product) string { return p.Name })
	fmt.Println(ok)
	// Output:
	// lamp true
	// false
}

func ExampleStack() {
	var s Stack[string] // zero value is ready to use
	s.Push("a")
	s.Push("b")
	top, _ := s.Pop()
	fmt.Println(top, s.Len())
	s.Pop()
	_, err := s.Pop()
	fmt.Println(err)
	// Output:
	// b 1
	// stack is empty
}

func ExampleCache_GetOrLoad() {
	calls := 0
	square := func(n int) (int, error) { calls++; return n * n, nil }
	c := NewCache[int, int]()
	c.GetOrLoad(4, square)
	v, _ := c.GetOrLoad(4, square)
	fmt.Println(v, "loaded", calls, "time(s)")
	// Output: 16 loaded 1 time(s)
}
