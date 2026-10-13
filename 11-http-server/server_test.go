package httpserver

import (
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

var quietLog = slog.New(slog.DiscardHandler)

// httptest.NewRecorder calls the handler directly: no network, very fast.
func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRoutes(t *testing.T) {
	h := NewHandler(quietLog)
	tests := []struct {
		name, method, path, body string
		wantStatus               int
		wantBody                 string
	}{
		{"path wildcard", "GET", "/hello/Gopher", "", 200, `"Hello, Gopher!"`},
		{"list", "GET", "/books?q=go", "", 200, "The Go Programming Language"},
		{"get one", "GET", "/books/1", "", 200, `"id":1`},
		{"bad id", "GET", "/books/abc", "", 400, "must be a number"},
		{"missing", "GET", "/books/99", "", 404, "not found"},
		{"method not allowed (automatic)", "PUT", "/books/1", "", 405, ""},
		{"create", "POST", "/books", `{"title":"Learning Go"}`, 201, `"title":"Learning Go"`},
		{"validation", "POST", "/books", `{"title":"  "}`, 400, "required"},
		{"unknown field", "POST", "/books", `{"titel":"x"}`, 400, "unknown field"},
		{"malformed", "POST", "/books", `{oops`, 400, "invalid JSON"},
		{"too large", "POST", "/books", `{"title":"` + strings.Repeat("x", 2<<20) + `"}`, 413, "too large"},
		{"static file", "GET", "/", "", 200, "embedded file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, h, tt.method, tt.path, tt.body)
			if rec.Code != tt.wantStatus || !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("%s %s = %d %q, want %d containing %q", tt.method, tt.path, rec.Code, rec.Body.String(), tt.wantStatus, tt.wantBody)
			}
		})
	}
}

func TestCreateSetsLocationAndUnsupportedMediaType(t *testing.T) {
	h := NewHandler(quietLog)
	rec := do(t, h, "POST", "/books", `{"title":"Go in Action"}`)
	if rec.Header().Get("Location") != "/books/2" {
		t.Fatalf("Location = %q", rec.Header().Get("Location"))
	}
	req := httptest.NewRequest("POST", "/books", strings.NewReader(`{}`)) // no Content-Type
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestMiddleware(t *testing.T) {
	h := NewHandler(quietLog)

	rec := do(t, h, "GET", "/panic", "")
	if rec.Code != 500 || !strings.Contains(rec.Body.String(), "internal server error") {
		t.Fatalf("panic not recovered: %d %s", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest("GET", "/hello/x", nil)
	req.Header.Set("X-Request-Id", "abc-123")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-Id") != "abc-123" {
		t.Fatal("request id not propagated")
	}
	if len(do(t, h, "GET", "/hello/x", "").Header().Get("X-Request-Id")) != 16 {
		t.Fatal("request id not generated")
	}
}

// A REAL server on a random port: graceful shutdown lets the in-flight request finish.
func TestGracefulShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- Serve(ctx, ln, NewHandler(quietLog), quietLog) }()

	resCh := make(chan *http.Response, 1)
	go func() {
		res, err := http.Get("http://" + ln.Addr().String() + "/slow?d=300ms")
		if err != nil {
			t.Error(err)
		}
		resCh <- res
	}()
	time.Sleep(100 * time.Millisecond) // the request is now in flight
	cancel()                           // like Ctrl+C

	res := <-resCh
	body, _ := io.ReadAll(res.Body)
	var got map[string]string
	json.Unmarshal(body, &got)
	if res.StatusCode != 200 || got["status"] != "done" {
		t.Fatalf("in-flight request was cut off: %d %s", res.StatusCode, body)
	}
	if err := <-served; err != nil {
		t.Fatalf("Serve returned %v", err)
	}
	if _, err := http.Get("http://" + ln.Addr().String() + "/hello/x"); err == nil {
		t.Fatal("server still accepting connections after shutdown")
	}
}
