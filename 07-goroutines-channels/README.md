# 07: Goroutines and channels

Concurrency is Go's signature feature. **Goroutines** are cheap (a few KB) concurrent functions scheduled by the
runtime over all CPU cores. **Channels** pass values between them safely.

## Concepts
- `go f()` starts a goroutine. `main` returning ends the program, including running goroutines.
- **`sync.WaitGroup`** waits for goroutines. `wg.Go(func(){...})` (Go 1.25) replaces `Add(1)` / `defer Done()`.
- **Channels**: `make(chan T)` (unbuffered: send and receive meet) vs `make(chan T, n)` (buffered).
  Direction types `<-chan T` / `chan<- T`. **The sender closes.** `for v := range ch` stops when closed.
- **Patterns**: pipeline (`Generate`, `Square`), fan-in (`Merge`), **worker pool** (fixed workers + job channel),
  and first-response-wins with timeout (`select` + `time.After`).
- **`select`**: wait on several channel operations, with timeouts and cancellation (`<-ctx.Done()`).
- **`errgroup`** (`golang.org/x/sync`): goroutines returning errors, a concurrency limit, and cancellation of the
  others on the first failure. This is what real services use for parallel calls.
- **Goroutine leaks**: a goroutine blocked forever on a channel nobody reads is never freed. Buffer the channel
  (`FirstOf`) or use a context.
- Shared memory needs synchronisation (next lesson). Here every goroutine writes its **own** slot.

## Run
```bash
go test -v -race ./07-goroutines-channels     # -race finds unsynchronised memory access
```
