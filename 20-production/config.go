// Package production shows what a Go service needs beyond working code: validated configuration,
// structured logs, metrics, health checks, profiling, graceful shutdown and a small container image.
package production

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

type Config struct {
	Port            int
	AdminPort       int // metrics + pprof on a SEPARATE port, never exposed publicly
	LogLevel        slog.Level
	LogFormat       string
	ShutdownTimeout time.Duration
	DrainDelay      time.Duration // time for load balancers to notice "not ready" before we stop accepting
}

// LoadConfig reads environment variables, applies defaults and reports EVERY problem at once (fail fast).
func LoadConfig(getenv func(string) string) (Config, error) {
	cfg := Config{Port: 8080, AdminPort: 9091, LogFormat: "json", ShutdownTimeout: 15 * time.Second, DrainDelay: 5 * time.Second}
	var errs []error

	intVar := func(key string, dst *int) {
		if v := getenv(key); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 || n > 65535 {
				errs = append(errs, fmt.Errorf("%s: %q is not a valid port", key, v))
				return
			}
			*dst = n
		}
	}
	durationVar := func(key string, dst *time.Duration) {
		if v := getenv(key); v != "" {
			d, err := time.ParseDuration(v)
			if err != nil || d < 0 {
				errs = append(errs, fmt.Errorf("%s: %q is not a duration like 10s", key, v))
				return
			}
			*dst = d
		}
	}

	intVar("PORT", &cfg.Port)
	intVar("ADMIN_PORT", &cfg.AdminPort)
	durationVar("SHUTDOWN_TIMEOUT", &cfg.ShutdownTimeout)
	durationVar("DRAIN_DELAY", &cfg.DrainDelay)
	if v := getenv("LOG_LEVEL"); v != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(v)); err != nil {
			errs = append(errs, fmt.Errorf("LOG_LEVEL: %q must be debug, info, warn or error", v))
		}
	}
	if v := getenv("LOG_FORMAT"); v != "" {
		if v != "json" && v != "text" {
			errs = append(errs, fmt.Errorf("LOG_FORMAT: %q must be json or text", v))
		}
		cfg.LogFormat = v
	}
	if cfg.Port != 0 && cfg.Port == cfg.AdminPort {
		errs = append(errs, errors.New("PORT and ADMIN_PORT must differ"))
	}
	return cfg, errors.Join(errs...)
}
