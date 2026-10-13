# 05: Generics

Type parameters (Go 1.18+) let you write a function or type once and use it with many types, without losing type safety.

## Concepts
- Generic functions: `func Map[T, R any](items []T, f func(T) R) []R`. Type arguments are usually **inferred**.
- **Constraints** describe the allowed types:
  - `any` (anything), `comparable` (usable with `==` and as map keys), `cmp.Ordered` (supports `<`).
  - Your own: `type Number interface { ~int | ~int64 | ~float64 }`. `~` includes named types like `type Rupees int`.
- Generic types: `Stack[T]`, `Cache[K comparable, V any]`. Methods use the type's parameters.
- `var zero T` gives the zero value of any type.
- The standard library is generic too: `slices`, `maps`, `cmp`, `sync.Map` alternatives, `iter.Seq[T]`.

## When to use them
Use generics for **containers and algorithms** that don't care about the element type (collections, caches, Map/Filter, retry helpers).
Don't use them when an interface already expresses the behaviour (`io.Reader`), or when there's only one type.
Start with concrete code and generalise when the duplication is real.

## Run
```bash
go test -v ./05-generics
```
