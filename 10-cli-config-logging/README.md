# 10: Command-line programs, configuration and logging

Go compiles to a single static binary, which makes it a favourite for CLIs and DevOps tools (Docker, Kubernetes,
Terraform and Hugo are written in Go).

## Concepts
- **`package main` + `func main()`**. `go build -o greeter ./10-cli-config-logging` produces one binary.
  Cross-compile with `GOOS=windows GOARCH=amd64 go build ...`.
- **Flags** with the `flag` package (`-name`, defaults, typed values, automatic `-h` help).
  `cobra` is the popular choice for big CLIs with subcommands.
- **Environment variables** for deployment configuration (`APP_GREETING`). Flags are for per-run options.
- **Validate config at startup** and fail with a clear message.
- **stdout vs stderr**, and **exit codes** (0 ok, 1 error, 2 usage error).
- **Structured logging** with `log/slog` (standard library): levels, key-value fields, `With(...)` for common fields,
  text or JSON handlers.
- **Signals**: `signal.NotifyContext` turns Ctrl+C / SIGTERM into context cancellation, which gives graceful shutdown.
- **Testable `main`**: keep `main` tiny and put the logic in `run(ctx, args, getenv, stdout, stderr) int`.
  Tests pass fake arguments, environment and buffers, with no subprocesses needed.

## Run
```bash
go run ./10-cli-config-logging -name Subbu -times 3
APP_GREETING=Namaste go run ./10-cli-config-logging -name Subbu -log-format json -verbose
go run ./10-cli-config-logging -watch          # Ctrl+C
go test -v ./10-cli-config-logging
```
