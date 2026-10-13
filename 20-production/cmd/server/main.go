// go run ./20-production/cmd/server
//
//	API:    http://localhost:8080/api/courses   health: /livez /readyz   version: /version
//	admin:  http://localhost:9091/metrics       profiling: /debug/pprof/
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	production "github.com/subrammiles/server_golang/20-production"
)

func main() {
	cfg, err := production.LoadConfig(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid configuration:\n"+err.Error())
		os.Exit(2)
	}
	log := production.NewLogger(os.Stdout, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	public, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		log.Error("listen", "error", err)
		os.Exit(1)
	}
	admin, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.AdminPort))
	if err != nil {
		log.Error("listen admin", "error", err)
		os.Exit(1)
	}

	app := production.NewApp(cfg, log, map[string]production.Check{
		// "database": func(ctx context.Context) error { return pool.Ping(ctx) },
	})
	if err := app.Run(ctx, public, admin); err != nil {
		log.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
