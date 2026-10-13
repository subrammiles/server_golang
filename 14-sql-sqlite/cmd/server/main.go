// Lesson 12's API on SQLite. Data survives restarts in courses.db.
//
//	go run ./14-sql-sqlite/cmd/server
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
	sqlitedb "github.com/subrammiles/server_golang/14-sql-sqlite"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := sqlitedb.Open(envOr("DB_FILE", "courses.db"))
	if err != nil {
		log.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if n, err := sqlitedb.Migrate(ctx, db); err != nil {
		log.Error("migrate", "error", err)
		os.Exit(1)
	} else if n > 0 {
		repo := sqlitedb.NewRepository(db)
		repo.ImportAll(ctx, course.Seed()) // first start: add sample data
	}

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Error("listen", "error", err)
		os.Exit(1)
	}
	if err := httpserver.Serve(ctx, ln, restapi.NewRouter(sqlitedb.NewRepository(db), log), log); err != nil {
		log.Error("server", "error", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
