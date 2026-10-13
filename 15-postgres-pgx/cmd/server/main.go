// Lesson 12's API on PostgreSQL.
//
//	docker compose -f 15-postgres-pgx/docker-compose.yml up -d --wait
//	DATABASE_URL=postgres://tutorial:tutorial@localhost:55432/courses go run ./15-postgres-pgx/cmd/server
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
	pgdb "github.com/subrammiles/server_golang/15-postgres-pgx"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Error("DATABASE_URL is not set")
		os.Exit(1)
	}
	pool, err := pgdb.Connect(ctx, url)
	if err != nil {
		log.Error("connect", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pgdb.Migrate(ctx, pool); err != nil {
		log.Error("migrate", "error", err)
		os.Exit(1)
	}
	var n int
	if pool.QueryRow(ctx, `SELECT COUNT(*) FROM courses`).Scan(&n); n == 0 {
		pgdb.Reset(ctx, pool)
	}

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Error("listen", "error", err)
		os.Exit(1)
	}
	if err := httpserver.Serve(ctx, ln, restapi.NewRouter(pgdb.NewRepository(pool), log), log); err != nil {
		log.Error("server", "error", err)
	}
}
