// go run ./11-http-server/cmd/server        then: curl localhost:8080/hello/Gopher  (Ctrl+C to stop gracefully)
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	httpserver "github.com/subrammiles/server_golang/11-http-server"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := ":" + envOr("PORT", "8080")
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("listen failed", "error", err)
		os.Exit(1)
	}
	if err := httpserver.Serve(ctx, ln, httpserver.NewHandler(log), log); err != nil {
		log.Error("server error", "error", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
