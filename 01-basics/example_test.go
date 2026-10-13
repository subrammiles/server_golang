package basics

import (
	"fmt"
	"strings"
)

// Example functions are runnable documentation: "go test" checks that the output matches.

func Example_variables() {
	var name string = "Go" // explicit type
	var version = 1.25     // type inferred (float64)
	year := 2009           // short declaration, only inside functions
	var zero int           // every type has a ZERO VALUE: 0, "", false, nil
	fmt.Println(name, version, year, zero)
	fmt.Printf("%T %T %T\n", name, version, year)
	// Output:
	// Go 1.25 2009 0
	// string float64 int
}

func Example_conversions() {
	n := 42
	f := float64(n) * 1.5 // no implicit conversion between numeric types
	s := fmt.Sprint(n)    // number -> string (string(n) would give the rune '*'!)
	fmt.Println(f, s+"!", RoleAdmin)
	// Output: 63 42! 2
}

func ExampleDivide() {
	if result, err := Divide(10, 3); err == nil { // "if" with an init statement
		fmt.Println(result)
	}
	_, err := Divide(1, 0)
	fmt.Println(err)
	// Output:
	// 3
	// cannot divide 1 by zero
}

func ExampleStats() {
	fmt.Println(Stats(4, 9, 1, 7))
	fmt.Println(Stats())
	// Output:
	// 1 9 21
	// 0 0 0
}

func Example_loops() {
	// "for" is Go's only loop keyword
	for i := 0; i < 3; i++ {
		fmt.Print(i, " ")
	}
	for i := range 3 { // range over an integer (Go 1.22)
		fmt.Print(i, " ")
	}
	n := 1
	for n < 100 { // while-style
		n *= 3
	}
	fmt.Println(n)
	// Output: 0 1 2 0 1 2 243
}

func ExampleGrade() {
	fmt.Println(Grade(95), Grade(80), Grade(10))
	// Output: A B F
}

func ExampleCounter() {
	next := Counter()
	next()
	next()
	fmt.Println(next())
	// Output: 3
}

func ExampleApply() {
	fmt.Println(Apply([]string{"go", "is", "fun"}, strings.ToUpper))
	// Output: [GO IS FUN]
}

func ExampleDeferOrder() {
	fmt.Println(DeferOrder())
	// Output: [body deferred 2 deferred 1]
}

func ExampleSwap() {
	a, b := 1, 2
	Swap(&a, &b) // & takes the address
	fmt.Println(a, b)
	// Output: 2 1
}
