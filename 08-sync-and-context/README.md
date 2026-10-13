# 08: Shared memory (sync) and context

## Shared memory
When goroutines touch the same variable and at least one writes, you need synchronisation. Otherwise you have a
**data race**: wrong results, randomly. `go run -race ./08-sync-and-context/race-demo` shows one.

| Tool | Use for |
|---|---|
| `sync.Mutex` | protect a group of fields. Keep check + update in **one** critical section |
| `sync.RWMutex` | many readers, rare writers |
| `sync/atomic` (`atomic.Int64`...) | simple counters and flags, lock-free |
| `sync.Once` | lazy one-time initialisation |
| channels | hand data from one goroutine to another (lesson 07) |

Always test with **`go test -race`**.

## Context
`context.Context` carries **cancellation**, **deadlines** and request-scoped **values** through a call chain.
- First parameter of every blocking function: `func Query(ctx context.Context, ...)`.
- `context.WithTimeout` / `WithCancel` / `WithDeadline`, and **always `defer cancel()`**.
- Check `<-ctx.Done()` in `select`. `ctx.Err()` is `context.Canceled` or `context.DeadlineExceeded`.
- HTTP servers give each request a context that is cancelled when the client disconnects (lesson 11).
- Values: only request-scoped data (request id, user), with a private key type. Never optional parameters.
- Background goroutines stop on `ctx.Done()`, otherwise they leak.

## Testing time: `testing/synctest` (Go 1.25)
Inside `synctest.Test`, time is fake: `time.Sleep(5 * time.Minute)` returns instantly, timers fire deterministically.
Timeouts and tickers are tested without slow or flaky sleeps.

## Run
```bash
go test -v -race ./08-sync-and-context
go run -race ./08-sync-and-context/race-demo
```
