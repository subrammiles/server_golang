# 03: Structs, methods and interfaces

Go has no classes and no inheritance. It has **structs** (data), **methods** (functions with a receiver),
**interfaces** (behaviour) and **embedding** (composition).

## Concepts
- **Structs** and composite literals: `Rect{Width: 2, Height: 3}`. Print with `%+v`.
- **Methods**: `func (r Rect) Area()` (value receiver, works on a copy) vs `func (r *Rect) Scale()` (pointer receiver, modifies the original).
- **Interfaces are satisfied implicitly**: any type with the methods qualifies. Keep them small
  (`io.Reader` has one method) and define them where they are **consumed**.
- `var _ Shape = Rect{}`: compile-time proof that a type implements an interface.
- **`fmt.Stringer`** (`String() string`) controls how a value prints.
- **Type assertion** `v.(Circle)` / `v, ok := ...`, and **type switch** `switch x := v.(type)`.
- `any` (= `interface{}`) holds any value. Prefer specific types or generics (lesson 05).
- **Embedding** promotes fields and methods. It's composition, not inheritance.
- Constructors are plain functions: `NewOrderService()`.
- **`io.Reader` / `io.Writer`**: the interfaces that make files, sockets, HTTP bodies and compression interchangeable (lesson 09).

## Run
```bash
go test -v ./03-types-and-interfaces
```
