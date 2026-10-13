# 17: Calling other services (robust HTTP clients)

Every backend calls other services. Over a network, everything fails eventually: slow, down, overloaded, rate-limited.

## Concepts
- **Never use `http.DefaultClient` / `http.Get` for service calls: they have no timeout.** Create one
  `*http.Client` with a `Timeout` and **reuse it** (its Transport pools keep-alive connections).
- **`http.NewRequestWithContext`**: the caller's context cancels the call (and the retries).
- **Always `defer res.Body.Close()`**, and drain it, or connections can't be reused. Limit how much of an error body you read.
- **Retry only transient failures** (network errors, 408/429/500/502/503/504), **never 4xx**.
- **Only retry idempotent requests**: GET yes, POST only with idempotency keys.
- **Exponential backoff with jitter**, and respect **`Retry-After`**.
- **Typed errors** (`*HTTPError`, `IsStatus`) so callers can choose to retry, fall back or return 502.
- **Concurrency limits** with `errgroup.SetLimit` (lesson 07).
- **Circuit breaker** (`Breaker`): fail fast while a dependency is down, then probe. `sony/gobreaker` is a production version.
- **Testing**: `httptest.NewServer` as a scripted stub (503, 503, then 200; slow responses). Fast, offline and deterministic.

## Run
```bash
go test -v -race ./17-http-clients
```
