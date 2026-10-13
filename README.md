# Go backend development: from scratch to advanced

A hands-on course in 20 lessons. Every lesson folder has:
- a **README.md** explaining the concepts, with commands to run,
- focused example code. Early lessons use Go's **Example tests** (runnable docs whose output `go test` checks),
  and later ones have runnable programs under `cmd/`,
- **tests** proving it all works.

Companion repositories teach the same backend ideas in other ecosystems: `server_nodejs`, `server_spring`,
`server_spring_boot`. The course catalog API (lesson 12) is the same domain in all of them, so you can compare.

## Requirements
- **Go 1.26+** (developed on Go 1.27; Go supports its two newest releases). One module for the whole repo: `go.mod` at the root.
- Docker (optional): PostgreSQL for lesson 15 (its tests are skipped without Docker) and the image in lesson 20.

```bash
go test ./...                    # every lesson
go test -race ./...              # with the race detector (recommended)
go test -v ./07-goroutines-channels
go vet ./...
```

## Roadmap

### Part 1: The language
| # | Lesson | You will learn |
|---|---|---|
| 01 | [Basics](01-basics) | variables, zero values, types, constants/iota, control flow, functions, closures, defer, pointers |
| 02 | [Collections](02-collections) | arrays vs slices (sharing gotchas), maps, strings/runes/UTF-8, `slices`/`maps`, iterators |
| 03 | [Types and interfaces](03-types-and-interfaces) | structs, value vs pointer receivers, implicit interfaces, type switches, embedding, `io.Reader` |
| 04 | [Errors](04-errors) | errors as values, wrapping `%w`, `errors.Is/As`, sentinel and custom errors, panic/recover |
| 05 | [Generics](05-generics) | type parameters, constraints, generic types, when (not) to use them |
| 06 | [Testing](06-testing) | table-driven tests, subtests, fakes, helpers, golden files, benchmarks, **fuzzing**, coverage |

### Part 2: Concurrency
| # | Lesson | You will learn |
|---|---|---|
| 07 | [Goroutines and channels](07-goroutines-channels) | `WaitGroup.Go`, channels, pipelines, fan-in, worker pools, `select`, timeouts, **errgroup** |
| 08 | [Sync and context](08-sync-and-context) | data races and `-race`, Mutex/RWMutex/atomic/Once, **context** cancellation, `testing/synctest` |

### Part 3: Standard library and tooling
| # | Lesson | You will learn |
|---|---|---|
| 09 | [I/O, files, JSON](09-io-files-json) | `io.Reader/Writer` composition, bufio, atomic file writes, JSON tags and custom marshalling, CSV, `go:embed` |
| 10 | [CLI, config, logging](10-cli-config-logging) | flags, env vars, exit codes, `slog`, signals, a testable `main` |

### Part 4: Building services
| # | Lesson | You will learn |
|---|---|---|
| 11 | [HTTP server](11-http-server) | `net/http` routing (Go 1.22 patterns), middleware, JSON safety, timeouts, **graceful shutdown**, `httptest` |
| 12 | [REST API](12-rest-api) | layered design, repository interface, validation, problem+json errors, paging, `html/template`, **contract tests** |
| 13 | [Gin and Echo](13-gin-and-echo) | the same API in two frameworks vs `net/http` (the old repo's examples, rebuilt) |
| 14 | [SQL with SQLite](14-sql-sqlite) | `database/sql`, pools, `ErrNoRows`, transactions, **SQL injection**, embedded migrations |
| 15 | [PostgreSQL + pgx](15-postgres-pgx) | pgxpool, `RETURNING`, joins, batches, **row locking under concurrency**, Testcontainers |
| 16 | [Auth and security](16-auth-security) | bcrypt, JWT, roles/ownership, rate limiting, security headers, CORS |
| 17 | [HTTP clients](17-http-clients) | timeouts, retries with backoff, idempotency, `Retry-After`, circuit breaker |
| 18 | [WebSockets](18-websockets) | realtime chat, rooms, presence, read/write loops, heartbeats |
| 19 | [gRPC](19-grpc) | Protocol Buffers, `buf`, unary and streaming RPCs, status codes, interceptors, deadlines, `bufconn` |
| 20 | [Production readiness](20-production) | config validation, JSON logs, **Prometheus metrics**, health checks, pprof, shutdown order, distroless Docker |

## What happened to the old code
The previous repo had separate modules with a few Gin/Echo examples, an empty `gorm` module and an empty `nosql` folder.
The Gin and Echo examples (ping, users, inline and template HTML) live on, tested, in [lesson 13](13-gin-and-echo).
Everything now builds as one module with `go test ./...`.

## Where to go next
`sqlc` (type-safe SQL code generation), `golangci-lint`, OpenTelemetry tracing, message queues (Kafka: see `server_kafka`,
NATS), Redis, Kubernetes operators (written in Go), and the book *The Go Programming Language*.
