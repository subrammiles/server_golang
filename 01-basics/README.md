# 01: Go basics

## Concepts
- **Packages**: every file starts with `package x`. `package main` + `func main()` makes a program.
  **Exported** names start with an upper-case letter.
- **Variables**: `var x int`, `var x = 1`, `x := 1` (inside functions). Every type has a **zero value** (`0`, `""`, `false`, `nil`).
- **Basic types**: `int`, `float64`, `string`, `bool`, `byte`, `rune`. **No implicit conversions**: `float64(n)`.
- **Constants** and `iota`.
- **Control flow**: `if` (with an init statement), `switch` (no fall-through), and `for`, the only loop
  (classic, while-style, `range 3`, `range slice`).
- **Functions**: multiple return values, named results, variadic `...int`, functions as values, **closures**.
- **`defer`**: runs when the function returns, last-in-first-out. Used for cleanup.
- **Pointers**: `&x` (address), `*p` (value). Go passes **copies**, so pass a pointer to modify.
- Tooling: `go run`, `go build`, `go test`, `go vet`, `gofmt` (one standard format, so no style debates).

## Run
```bash
go test -v ./01-basics          # runs the Example functions and checks their output
go doc ./01-basics Divide       # docs come from comments
```
