// go run ./12-rest-api/cmd/server     then open http://localhost:8080 and http://localhost:8080/api/courses
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	httpserver "github.com/subrammiles/server_golang/11-http-server"
	restapi "github.com/subrammiles/server_golang/12-rest-api"
	"github.com/subrammiles/server_golang/12-rest-api/course"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Error("listen", "error", err)
		os.Exit(1)
	}
	handler := httpserver.Chain(restapi.NewRouter(course.NewMemoryRepository(course.Seed()...), log),
		httpserver.RequestID, httpserver.Logger(log), httpserver.Recover(log)) // reuse lesson 11's middleware
	if err := httpserver.Serve(ctx, ln, handler, log); err != nil {
		log.Error("server", "error", err)
		os.Exit(1)
	}
}
