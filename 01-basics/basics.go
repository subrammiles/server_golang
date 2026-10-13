// Package basics covers the building blocks of Go: variables, types, functions, closures and defer.
// Run the examples: go test -v ./01-basics
package basics

import (
	"fmt"
	"strings"
)

// Constants are fixed at compile time. iota numbers related constants automatically.
const (
	RoleGuest = iota // 0
	RoleUser         // 1
	RoleAdmin        // 2
)

// Exported names start with an UPPER-CASE letter (visible to other packages); lower-case = private.
const MaxUsers = 100

// Divide returns TWO values: Go functions often return a result plus an error (lesson 04).
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, fmt.Errorf("cannot divide %d by zero", a)
	}
	return a / b, nil
}

// Stats uses NAMED results; a bare "return" returns their current values.
func Stats(numbers ...int) (minimum, maximum, sum int) { // ...int = variadic: any number of ints
	if len(numbers) == 0 {
		return
	}
	minimum, maximum = numbers[0], numbers[0]
	for _, n := range numbers {
		minimum = min(minimum, n) // built-in min/max since Go 1.21
		maximum = max(maximum, n)
		sum += n
	}
	return
}

// Grade shows if/else and switch. Go's switch needs no "break" and cases can be expressions.
func Grade(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 75:
		return "B"
	case score >= 50:
		return "C"
	default:
		return "F"
	}
}

// Counter returns a CLOSURE: a function that remembers the variable it captured.
func Counter() func() int {
	count := 0
	return func() int {
		count++
		return count
	}
}

// Apply takes a function as a parameter: functions are values in Go.
func Apply(words []string, transform func(string) string) []string {
	out := make([]string, 0, len(words))
	for _, w := range words {
		out = append(out, transform(w))
	}
	return out
}

// DeferOrder shows that deferred calls run when the function returns, in LIFO order.
// Typical use: defer file.Close(), defer mu.Unlock(), defer cancel().
func DeferOrder() []string {
	var steps []string
	record := func(s string) { steps = append(steps, s) }
	func() {
		defer record("deferred 1")
		defer record("deferred 2")
		record("body")
	}()
	return steps
}

// Swap shows pointers: a function receives a COPY of its arguments; pass a pointer to modify the original.
func Swap(a, b *int) {
	*a, *b = *b, *a
}

// Shout demonstrates the strings package.
func Shout(s string) string {
	return strings.ToUpper(strings.TrimSpace(s)) + "!"
}
