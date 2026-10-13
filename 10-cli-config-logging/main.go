// A small command-line tool showing flags, environment config, structured logging, exit codes and signals.
//
//	go run ./10-cli-config-logging -name Subbu -times 3
//	go run ./10-cli-config-logging -log-format json -verbose
//	APP_GREETING=Namaste go run ./10-cli-config-logging -name Subbu
//	go run ./10-cli-config-logging -watch       (Ctrl+C to stop gracefully)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	Name      string
	Times     int
	Greeting  string
	LogFormat string
	Verbose   bool
	Watch     bool
}

// parseConfig reads FLAGS (per run) and ENVIRONMENT VARIABLES (per deployment). Taking args and a
// getenv function (instead of os.Args / os.Getenv directly) makes it testable.
func parseConfig(args []string, getenv func(string) string) (Config, error) {
	cfg := Config{Greeting: "Hello"}
	if g := getenv("APP_GREETING"); g != "" {
		cfg.Greeting = g
	}

	fs := flag.NewFlagSet("greeter", flag.ContinueOnError)
	fs.StringVar(&cfg.Name, "name", "world", "who to greet")
	fs.IntVar(&cfg.Times, "times", 1, "how many times")
	fs.StringVar(&cfg.LogFormat, "log-format", "text", "text or json")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "debug logging")
	fs.BoolVar(&cfg.Watch, "watch", false, "keep running until interrupted")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if cfg.Times < 1 || cfg.Times > 10 {
		return cfg, fmt.Errorf("-times must be between 1 and 10, got %d", cfg.Times)
	}
	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		return cfg, fmt.Errorf("-log-format must be text or json")
	}
	return cfg, nil
}

// newLogger builds a STRUCTURED logger (log/slog): key=value pairs or JSON instead of free text.
func newLogger(w io.Writer, cfg Config) *slog.Logger {
	level := slog.LevelInfo
	if cfg.Verbose {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}
	if cfg.LogFormat == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

// run contains the program; main only wires it to the real OS. Returning an exit code keeps run testable.
func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	cfg, err := parseConfig(args, getenv)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err) // errors to STDERR
		return 2                            // non-zero exit code = failure (2 = usage error by convention)
	}

	logger := newLogger(stderr, cfg).With("app", "greeter") // fields attached to every line
	logger.Debug("config loaded", "name", cfg.Name, "times", cfg.Times)

	for i := range cfg.Times {
		fmt.Fprintf(stdout, "%s, %s! (%d)\n", cfg.Greeting, cfg.Name, i+1) // results to STDOUT
	}
	logger.Info("greeted", "name", cfg.Name, "count", cfg.Times)

	if cfg.Watch {
		logger.Info("watching; press Ctrl+C to stop")
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				logger.Info("still running")
			case <-ctx.Done(): // cancelled by SIGINT/SIGTERM
				logger.Info("shutting down cleanly")
				return 0
			}
		}
	}
	return 0
}

func main() {
	// NotifyContext cancels ctx on Ctrl+C (SIGINT) or SIGTERM (what Docker/Kubernetes send)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}
