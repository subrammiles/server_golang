package production

import (
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func NewLogger(w io.Writer, cfg Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	var h slog.Handler = slog.NewJSONHandler(w, opts) // JSON: searchable in ELK, Loki, Datadog, CloudWatch
	if cfg.LogFormat == "text" {
		h = slog.NewTextHandler(w, opts)
	}
	return slog.New(h).With("service", "course-api", "version", Version())
}

// Version reports the build: set at build time with
// go build -ldflags "-X github.com/subrammiles/server_golang/20-production.version=1.2.3"
var version = ""

func Version() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok { // VCS revision is embedded automatically by `go build`
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && len(s.Value) >= 7 {
				return s.Value[:7]
			}
		}
	}
	return "dev"
}

// Metrics for Prometheus: the RED method = Rate, Errors, Duration of every endpoint.
type Metrics struct {
	Registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
}

func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry() // own registry instead of the global one: isolated and testable
	m := &Metrics{
		Registry: reg,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total", Help: "HTTP requests by route and status.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_request_duration_seconds", Help: "HTTP request latency.", Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{Name: "http_requests_in_flight", Help: "Requests being served."}),
	}
	reg.MustRegister(m.requests, m.duration, m.inFlight,
		collectors.NewGoCollector(),                                       // goroutines, GC, memory
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})) // CPU, open files
	return m
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Instrument records metrics and an access log line per request, then calls next.
// It labels by ROUTE PATTERN ("GET /api/courses/{id}", looked up in mux), never the raw path:
// one label value per id would explode the number of time series.
// next must already include panic recovery, so panicking requests are still counted (as 500s).
func (m *Metrics) Instrument(log *slog.Logger, mux *http.ServeMux, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.inFlight.Inc()
		defer m.inFlight.Dec()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		_, route := mux.Handler(r)
		if route == "" {
			route = "unmatched"
		}
		m.requests.WithLabelValues(r.Method, route, strconv.Itoa(sw.status)).Inc()
		m.duration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
		level := slog.LevelInfo
		if sw.status >= 500 {
			level = slog.LevelError
		}
		log.Log(r.Context(), level, "request", "method", r.Method, "route", route, "status", sw.status, "duration_ms", time.Since(start).Milliseconds())
	})
}
