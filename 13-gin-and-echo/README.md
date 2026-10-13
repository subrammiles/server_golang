# 13: Web frameworks: Gin and Echo

The old version of this repo started with Gin and Echo "hello world" examples. They're all here (`/ping`,
`/users`, inline and template HTML), now with tests, and both frameworks also serve lesson 12's course service.

## Same API, three ways
| | `net/http` (lessons 11-12) | Gin | Echo |
|---|---|---|---|
| route | `mux.HandleFunc("GET /x/{id}", h)` | `r.GET("/x/:id", h)` | `e.GET("/x/:id", h)` |
| path param | `r.PathValue("id")` | `c.Param("id")` | `c.Param("id")` |
| JSON out | your `writeJSON` helper | `c.JSON(200, gin.H{...})` | `return c.JSON(200, v)` |
| JSON in + validation | decode + validate by hand | `c.ShouldBindJSON(&v)` + `binding` tags | `c.Bind(&v)` + your validator |
| errors | write the response | write the response | **return** an error |
| groups / middleware | `Chain(...)` | `r.Group`, `r.Use` | `e.Group`, `e.Use` |

## Should I use a framework?
- Since Go 1.22, **`net/http` covers most needs**: method routing, wildcards, middleware. Many teams use it alone,
  maybe with small helper packages.
- Frameworks add convenience: binding and validation, route groups, a large middleware ecosystem, less boilerplate.
- All of them produce an `http.Handler`, so **testing is identical** (`httptest`, one table for both frameworks).
- Others you'll meet: chi (a small router on top of net/http), Fiber (Express-like, built on fasthttp, not net/http compatible).

## Run
```bash
go run ./13-gin-and-echo/cmd/gin       # curl localhost:8080/ping
go run ./13-gin-and-echo/cmd/echo      # curl localhost:1323/ping
go test -v ./13-gin-and-echo
```
