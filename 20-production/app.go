package production

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	httpserver "github.com/subrammiles/server_golang/11-http-server"
	"github.com/subrammiles/server_golang/12-rest-api/course"
)

// Check is a readiness dependency check (database ping, downstream service...).
type Check func(ctx context.Context) error

type App struct {
	cfg     Config
	log     *slog.Logger
	metrics *Metrics
	ready   atomic.Bool
	checks  map[string]Check
	Public  http.Handler
	Admin   http.Handler
}

func NewApp(cfg Config, log *slog.Logger, checks map[string]Check) *App {
	a := &App{cfg: cfg, log: log, metrics: NewMetrics(), checks: checks}

	mux := http.NewServeMux()
	// LIVENESS: is the process healthy? Failing it makes Kubernetes RESTART the container.
	// Keep it dependency-free, or a database outage restarts every replica at once.
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	// READINESS: may we receive traffic now? Failing it only removes us from the load balancer.
	mux.HandleFunc("GET /readyz", a.readyz)
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"version": Version()})
	})
	// Lesson 12's API, registered on THIS mux so metrics can see each route's exact pattern
	svc := course.NewService(course.NewMemoryRepository(course.Seed()...))
	course.NewHandler(svc, log).Register(mux)
	mux.HandleFunc("GET /api/boom", func(http.ResponseWriter, *http.Request) { panic("unexpected") })

	// Order matters: RequestID (outermost) -> metrics/logging -> panic recovery -> routes.
	// Recovery sits INSIDE the instrumentation, so a panic is still measured and logged as a 500.
	a.Public = httpserver.RequestID(a.metrics.Instrument(log, mux, httpserver.Recover(log)(mux)))

	admin := http.NewServeMux()
	admin.Handle("GET /metrics", promhttp.HandlerFor(a.metrics.Registry, promhttp.HandlerOpts{}))
	// PROFILING: CPU, heap, goroutines. Use: go tool pprof http://localhost:9091/debug/pprof/profile?seconds=10
	admin.HandleFunc("/debug/pprof/", pprof.Index)
	admin.HandleFunc("/debug/pprof/profile", pprof.Profile)
	admin.HandleFunc("/debug/pprof/trace", pprof.Trace)
	a.Admin = admin

	a.ready.Store(true)
	return a
}

func (a *App) readyz(w http.ResponseWriter, r *http.Request) {
	if !a.ready.Load() {
		http.Error(w, "shutting down", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	results := map[string]string{}
	healthy := true
	for name, check := range a.checks {
		if err := check(ctx); err != nil {
			results[name], healthy = err.Error(), false
		} else {
			results[name] = "ok"
		}
	}
	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"ready": healthy, "checks": results})
}

// Run serves until ctx is cancelled (SIGTERM), then shuts down in the order that loses no requests:
// 1. fail readiness  2. wait DrainDelay so load balancers stop routing to us
// 3. srv.Shutdown: stop accepting, finish in-flight requests  4. close resources.
func (a *App) Run(ctx context.Context, public, admin net.Listener) error {
	pub := &http.Server{Handler: a.Public, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	adm := &http.Server{Handler: a.Admin, ReadHeaderTimeout: 5 * time.Second}

	errCh := make(chan error, 2)
	go func() { errCh <- pub.Serve(public) }()
	go func() { errCh <- adm.Serve(admin) }()
	a.log.Info("started", "public", public.Addr().String(), "admin", admin.Addr().String())

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	a.log.Info("shutdown: draining", "delay", a.cfg.DrainDelay)
	a.ready.Store(false)
	time.Sleep(a.cfg.DrainDelay)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()
	err := errors.Join(pub.Shutdown(shutdownCtx), adm.Shutdown(shutdownCtx))
	// close database pools, flush tracers, etc. here
	a.log.Info("shutdown complete", "error", err)
	return err
}
