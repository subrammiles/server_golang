# 12: A production-style REST API

The course catalog (the same domain as the Spring Boot and Node.js repos), structured the way Go services usually are.

```
12-rest-api/
  app.go                    NewRouter(repo, log): the composition root
  cmd/server/main.go        main: config, server, graceful shutdown (reuses lesson 11)
  course/
    course.go               domain types, errors, validation, the Repository interface
    service.go              business rules (unique titles, PUT vs PATCH), no HTTP, no SQL
    memory.go               in-memory Repository
    handler.go              HTTP adapter + error -> status mapping (problem+json)
    coursetest/contract.go  tests every Repository implementation must pass
  web/                      server-side HTML with html/template (from the old gin template example)
```

## Concepts
- **Package by feature** (`course/`), with **small interfaces defined by the consumer** (`Repository`).
- **Layers**: handler, then service, then repository. Each is testable alone, and storage can be swapped (lessons 14 and 15).
- **Dependency injection** by passing values (`NewService(repo)`). No framework needed.
- **Validation** returning every invalid field. **Errors to HTTP status in one place** (400/404/409/500), as RFC 9457 problem+json.
- **PATCH with pointer fields**: `nil` = not sent, which is different from "set to 0".
- **Paging, filtering and sorting** with a whitelist of sort fields.
- **Context** (`r.Context()`) passed all the way down to the repository.
- **`html/template`**: auto-escaping prevents XSS (tested). `//go:embed` templates and parse them once.
- **Contract tests** (`coursetest.RunContract`): one suite run against every repository implementation.
- **Money as integers** (`Price int64`), never floats.

## Run
```bash
go run ./12-rest-api/cmd/server
curl 'localhost:8080/api/courses?category=backend&sort=-price'
curl -i -X POST localhost:8080/api/courses -d '{"title":"Docker","category":"devops","durationHours":6,"price":799}'
curl -i -X POST localhost:8080/api/courses -d '{"title":""}'
open http://localhost:8080/
go test -v ./12-rest-api/...
```
