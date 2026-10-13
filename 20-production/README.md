# 20: Production readiness

Working code isn't enough. In production you must **configure, observe, debug, deploy and stop** the service safely.
This lesson wraps lesson 12's API with all of that.

## Concepts
- **Configuration from env vars**, validated at startup with **every** error reported (`errors.Join`). Exit code 2 when invalid.
- **Structured JSON logs** (`slog`) with service and version on every line, one access line per request, errors at ERROR.
- **Metrics** with Prometheus (`client_golang`), the **RED** method: request Rate, Errors and Duration (histogram),
  plus Go runtime and process metrics. Label by **route pattern** (`GET /api/courses/{id}`), never the raw path
  (high cardinality would explode the number of series). Grafana dashboards and alerts build on these.
- **Health**: `/livez` (restart me if failing; no dependency checks) vs `/readyz` (route traffic to me? checks DB etc.).
- **Admin port** for `/metrics` and **pprof** (CPU, heap and goroutine profiles: `go tool pprof`), never on the public port.
- **Version** from `-ldflags -X` or the VCS info Go embeds automatically (`debug.ReadBuildInfo`).
- **Graceful shutdown in the right order**: fail readiness, **drain delay** (load balancers notice), `Shutdown`
  (finish in-flight requests), then close resources. The test checks every step.
- **Container**: multi-stage build, `CGO_ENABLED=0` static binary, `-trimpath -ldflags "-s -w"`, **distroless**
  `nonroot` base image. The result is a few MB with no shell and no root.
- More to explore: OpenTelemetry tracing, `GOMEMLIMIT` / `GOMAXPROCS` in containers, `govulncheck`,
  `golangci-lint`, CI (`go vet`, `go test -race`), and profile-guided optimisation (PGO).

## Run
```bash
go run ./20-production/cmd/server
curl localhost:8080/api/courses ; curl localhost:8080/readyz ; curl localhost:8080/version
curl -s localhost:9091/metrics | grep http_requests_total
LOG_LEVEL=loud PORT=http go run ./20-production/cmd/server     # refuses to start, lists both problems
docker build -f 20-production/Dockerfile --build-arg VERSION=1.0.0 -t go-production . && docker run -p 8080:8080 -p 9091:9091 go-production
go test -v ./20-production
```
