# 02: Collections: arrays, slices, maps, strings

## Concepts
- **Arrays** `[3]int` have a fixed size. **Slices** `[]int` are what you use: a pointer + length + capacity over an array.
  - `append` may move data to a new array, so always assign the result: `s = append(s, x)`.
  - **Sub-slices share memory** with the original. Use `slices.Clone` for an independent copy.
  - `make([]T, len, cap)` pre-sizes slices.
- **Maps** `map[K]V`: `m[k]` returns the zero value when missing. Use `v, ok := m[k]` to tell. `delete(m, k)`.
  Iteration order is **random**, so sort the keys if order matters. A nil map can be read but not written.
- **Strings** are immutable **UTF-8 bytes**. `len(s)` counts bytes. A `rune` is a character (`[]rune(s)`, `utf8.RuneCountInString`).
  Build strings with `strings.Builder`.
- Standard packages `slices`, `maps`, `cmp`: `Sort`, `SortFunc`, `Contains`, `BinarySearch`, `Max`, `Collect`, `Keys`...
- **Iterators** (`iter.Seq`, Go 1.23): functions you `range` over, producing values lazily.

## Run
```bash
go test -v ./02-collections
```
