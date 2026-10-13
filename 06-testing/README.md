# 06: Testing

Testing is built into Go: no framework to install. Files ending in `_test.go`, functions `TestXxx(t *testing.T)`.

## Concepts
- **Table-driven tests** with **subtests** (`t.Run`): the standard Go style. Run one case with `-run 'TestSlugify/digits'`.
- `t.Errorf` (report, continue) vs `t.Fatalf` (report, stop). `t.Parallel()`.
- **Helpers** with `t.Helper()`, including a generic `assertEqual`. Libraries like `testify` exist, but the standard library is enough.
- **Fakes** through interfaces (`Notifier`): no mocking framework needed.
- `t.TempDir()`, `t.Cleanup()`.
- **Golden files** in `testdata/` with an `-update` flag.
- **Examples** (`ExampleXxx` + `// Output:`), used in lessons 01–05: docs that are tested.
- **Benchmarks** with `b.Loop()` (`-bench=. -benchmem`).
- **Fuzzing**: random inputs checked against *properties*. Failing inputs are saved in `testdata/fuzz/` and become regression tests.
- **Coverage**: `go test -cover ./...`, or `-coverprofile=c.out` then `go tool cover -html=c.out`.
- **Race detector**: `go test -race ./...` (lesson 08).
- HTTP handlers are tested with `net/http/httptest` (lesson 11 onwards).

## Run
```bash
go test -v ./06-testing
go test -run 'TestSlugify/unicode' -v ./06-testing
go test -bench=. -benchmem ./06-testing
go test -fuzz=FuzzSlugify -fuzztime=10s ./06-testing
go test -cover ./...
```
