# 04: Errors

Go has no exceptions for normal failures. **Errors are values** returned as the last result, and you check them right away:

```go
user, err := repo.Find(id)
if err != nil {
    return fmt.Errorf("load user %d: %w", id, err)   // add context, keep the cause
}
```

## Concepts
- The `error` interface: anything with `Error() string`.
- `errors.New` and `fmt.Errorf`. **Wrap with `%w`** to add context ("transfer failed: account "carol": not found").
- **Sentinel errors** (`ErrNotFound`) compared with **`errors.Is`**, which sees through wrapping.
- **Custom error types** (`*ValidationError`) carrying data, extracted with **`errors.As`**.
- `errors.Join` to return several errors at once.
- **`panic` / `recover`**: for bugs and truly impossible states, not for business errors. HTTP servers recover
  panics per request (lesson 11).
- Handle each error **once**: either return it (with context) or log it. Not both.
- Don't ignore errors (`_ = f()`) unless you can explain why. `go vet` and linters help.

## Run
```bash
go test -v ./04-errors
```
