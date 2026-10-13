# 11: HTTP servers with the standard library

Many Go services use **only `net/http`**. Since Go 1.22 its router supports HTTP methods and path wildcards.

## Concepts
- `http.Handler` (`ServeHTTP(w, r)`) and `http.HandlerFunc`. Every request runs in **its own goroutine**,
  so **protect shared state** (`sync.Mutex` in `store`).
- **Routing** with `http.NewServeMux()`: `"GET /books/{id}"`, `r.PathValue("id")`, query via `r.URL.Query()`.
  Wrong method gives an automatic **405**, an unknown path a **404**.
- **JSON in and out** (`readJSON` / `writeJSON`): `MaxBytesReader` (413), Content-Type check (415),
  `DisallowUnknownFields`, correct status codes, `Location` on 201.
- **Middleware** = `func(http.Handler) http.Handler`: request ids (stored in the request **context**),
  structured logging (`slog`), and **panic recovery** so one bad request can't crash the server. `Chain` composes them.
- **Static files** with `http.FileServerFS` over an **embedded** filesystem.
- **`http.Server` timeouts**: the defaults wait forever. Always set `ReadHeaderTimeout` and friends.
- **Graceful shutdown**: `signal.NotifyContext` + `srv.Shutdown(ctx)` finishes in-flight requests.
  `r.Context()` lets handlers stop work when the client disconnects.
- **Testing**: `httptest.NewRecorder` (call the handler directly) and a real server on port 0 for shutdown behaviour.

## Run
```bash
go run ./11-http-server/cmd/server
curl localhost:8080/hello/Gopher
curl -i -X POST localhost:8080/books -H 'Content-Type: application/json' -d '{"title":"Learning Go"}'
curl -i -X PUT localhost:8080/books/1                  # 405
curl localhost:8080/panic                              # 500, the server keeps running
go test -v ./11-http-server
```
