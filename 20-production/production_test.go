package production

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestConfigDefaultsAndOverrides(t *testing.T) {
	cfg, err := LoadConfig(env(map[string]string{"PORT": "3000", "LOG_LEVEL": "debug", "DRAIN_DELAY": "1s"}))
	if err != nil || cfg.Port != 3000 || cfg.AdminPort != 9091 || cfg.LogLevel != slog.LevelDebug || cfg.DrainDelay != time.Second {
		t.Fatalf("%+v %v", cfg, err)
	}
}

func TestConfigReportsEveryProblem(t *testing.T) {
	_, err := LoadConfig(env(map[string]string{"PORT": "http", "LOG_LEVEL": "loud", "SHUTDOWN_TIMEOUT": "soon", "LOG_FORMAT": "xml"}))
	for _, want := range []string{"PORT", "LOG_LEVEL", "SHUTDOWN_TIMEOUT", "LOG_FORMAT"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %s: %v", want, err)
		}
	}
}

func newTestApp(t *testing.T, checks map[string]Check) (*App, *bytes.Buffer) {
	var logs bytes.Buffer
	cfg, _ := LoadConfig(env(nil))
	cfg.DrainDelay = 50 * time.Millisecond
	return NewApp(cfg, NewLogger(&logs, cfg), checks), &logs
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestHealthEndpoints(t *testing.T) {
	dbUp := true
	app, _ := newTestApp(t, map[string]Check{"database": func(context.Context) error {
		if !dbUp {
			return io.ErrUnexpectedEOF
		}
		return nil
	}})
	if get(app.Public, "/livez").Code != 200 || get(app.Public, "/readyz").Code != 200 {
		t.Fatal("should be live and ready")
	}
	dbUp = false
	if rec := get(app.Public, "/readyz"); rec.Code != 503 || !strings.Contains(rec.Body.String(), "unexpected EOF") {
		t.Fatalf("readyz with DB down: %d %s", rec.Code, rec.Body)
	}
	if get(app.Public, "/livez").Code != 200 {
		t.Fatal("liveness must not depend on the database")
	}
}

func TestMetricsAndLogs(t *testing.T) {
	app, logs := newTestApp(t, nil)
	get(app.Public, "/api/courses/1")
	get(app.Public, "/api/courses/2")
	get(app.Public, "/api/courses/999")
	get(app.Public, "/api/boom") // panic -> recovered -> 500

	metrics := get(app.Admin, "/metrics").Body.String()
	for _, want := range []string{
		`http_requests_total{method="GET",route="GET /api/courses/{id}",status="200"} 2`, // labelled by PATTERN, not raw path
		`http_requests_total{method="GET",route="GET /api/courses/{id}",status="404"} 1`,
		`http_requests_total{method="GET",route="GET /api/boom",status="500"} 1`, // panics are counted too
		`http_request_duration_seconds_bucket`,
		`go_goroutines`,
	} {
		if !strings.Contains(metrics, want) {
			t.Errorf("metrics missing %s", want)
		}
	}

	var line map[string]any
	if err := json.Unmarshal([]byte(strings.Split(logs.String(), "\n")[0]), &line); err != nil {
		t.Fatalf("logs are not JSON: %s", logs)
	}
	if line["service"] != "course-api" || line["route"] != "GET /api/courses/{id}" {
		t.Fatalf("log line %v", line)
	}
	if !strings.Contains(logs.String(), `"level":"ERROR"`) {
		t.Fatal("the panic should be logged at ERROR")
	}
}

func TestPprofIsOnlyOnTheAdminPort(t *testing.T) {
	app, _ := newTestApp(t, nil)
	if get(app.Admin, "/debug/pprof/").Code != 200 {
		t.Fatal("pprof missing on admin port")
	}
	if get(app.Public, "/debug/pprof/").Code == 200 || get(app.Public, "/metrics").Code == 200 {
		t.Fatal("pprof/metrics must not be public")
	}
}

func TestGracefulShutdownDrainsThenFinishesInFlightRequests(t *testing.T) {
	app, _ := newTestApp(t, nil)
	public, _ := net.Listen("tcp", "127.0.0.1:0")
	admin, _ := net.Listen("tcp", "127.0.0.1:0")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx, public, admin) }()
	base := "http://" + public.Addr().String()

	waitUntil(t, func() bool { res, err := http.Get(base + "/livez"); return err == nil && res.StatusCode == 200 })
	cancel() // SIGTERM

	// during the drain delay: still serving, but no longer ready
	waitUntil(t, func() bool { res, err := http.Get(base + "/readyz"); return err == nil && res.StatusCode == 503 })
	if res, err := http.Get(base + "/api/courses"); err != nil || res.StatusCode != 200 {
		t.Fatalf("requests during drain should still succeed: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("Run returned %v", err)
	}
	if _, err := http.Get(base + "/livez"); err == nil {
		t.Fatal("still accepting after shutdown")
	}
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	for range 100 {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition never became true")
}
