package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

// stub is a scripted fake of the remote API: offline, deterministic, can fail on demand.
type stub struct {
	mu      sync.Mutex
	replies map[string][]func(http.ResponseWriter)
	hits    map[string]int
}

func newStub(t *testing.T) (*stub, *httptest.Server) {
	s := &stub{replies: map[string][]func(http.ResponseWriter){}, hits: map[string]int{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		s.mu.Lock()
		s.hits[key]++
		queue := s.replies[key]
		var reply func(http.ResponseWriter)
		switch {
		case len(queue) == 0:
			reply = status(404, `{"error":"no script"}`)
		case len(queue) == 1:
			reply = queue[0]
		default:
			reply, s.replies[key] = queue[0], queue[1:]
		}
		s.mu.Unlock()
		reply(w)
	}))
	t.Cleanup(srv.Close)
	return s, srv
}

func (s *stub) script(key string, replies ...func(http.ResponseWriter)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replies[key] = replies
}

func (s *stub) count(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits[key]
}

func status(code int, body string, headers ...string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		for i := 0; i+1 < len(headers); i += 2 {
			w.Header().Set(headers[i], headers[i+1])
		}
		w.WriteHeader(code)
		fmt.Fprint(w, body)
	}
}

func newClient(url string) *Client {
	c := New(url, 500*time.Millisecond)
	c.BaseDelay = time.Millisecond
	return c
}

func TestRetriesTransientErrors(t *testing.T) {
	s, srv := newStub(t)
	s.script("GET /flaky", status(503, ""), status(502, ""), status(200, `{"ok":true}`))
	var out struct{ OK bool }
	if err := newClient(srv.URL).Get(context.Background(), "/flaky", &out); err != nil || !out.OK {
		t.Fatalf("got %v %v", out, err)
	}
	if s.count("GET /flaky") != 3 {
		t.Fatalf("%d attempts", s.count("GET /flaky"))
	}
}

func TestGivesUpAndReportsTheStatus(t *testing.T) {
	s, srv := newStub(t)
	s.script("GET /down", status(500, "boom"))
	err := newClient(srv.URL).Get(context.Background(), "/down", nil)
	if !IsStatus(err, 500) || s.count("GET /down") != 3 {
		t.Fatalf("err %v after %d attempts", err, s.count("GET /down"))
	}
}

func TestClientErrorsAreNotRetried(t *testing.T) {
	s, srv := newStub(t)
	err := newClient(srv.URL).Get(context.Background(), "/missing", nil)
	if !IsStatus(err, 404) || s.count("GET /missing") != 1 {
		t.Fatalf("err %v, %d attempts", err, s.count("GET /missing"))
	}
}

func TestPostIsNotRetriedUnlessIdempotent(t *testing.T) {
	s, srv := newStub(t)
	s.script("POST /payments", status(503, ""), status(201, `{"id":9}`))
	c := newClient(srv.URL)
	if err := c.Post(context.Background(), "/payments", map[string]int{"amount": 10}, nil, false); !IsStatus(err, 503) {
		t.Fatalf("got %v", err)
	}
	var out struct{ ID int }
	if err := c.Post(context.Background(), "/payments", map[string]int{"amount": 10}, &out, true); err != nil || out.ID != 9 {
		t.Fatalf("got %v %v", out, err)
	}
}

func TestHonoursRetryAfter(t *testing.T) {
	s, srv := newStub(t)
	s.script("GET /limited", status(429, "", "Retry-After", "1"), status(200, `{}`))
	start := time.Now()
	if err := newClient(srv.URL).Get(context.Background(), "/limited", nil); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < 950*time.Millisecond {
		t.Fatal("did not wait for Retry-After")
	}
}

func TestTimeoutAndCallerCancellation(t *testing.T) {
	s, srv := newStub(t)
	s.script("GET /slow", func(w http.ResponseWriter) { time.Sleep(300 * time.Millisecond); w.WriteHeader(200) })
	c := New(srv.URL, 50*time.Millisecond)
	c.Retries = 0
	if err := c.Get(context.Background(), "/slow", nil); err == nil {
		t.Fatal("want timeout")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := New(srv.URL, time.Second).Get(ctx, "/slow", nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}

func TestParallelCallsWithALimit(t *testing.T) {
	s, srv := newStub(t)
	for i := range 6 {
		s.script(fmt.Sprintf("GET /users/%d", i), status(200, fmt.Sprintf(`{"id":%d}`, i)))
	}
	c := newClient(srv.URL)
	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(3) // be polite: at most 3 requests in flight
	ids := make([]int, 6)
	for i := range 6 {
		g.Go(func() error {
			var u struct{ ID int }
			err := c.Get(ctx, fmt.Sprintf("/users/%d", i), &u)
			ids[i] = u.ID
			return err
		})
	}
	if err := g.Wait(); err != nil || ids[5] != 5 {
		t.Fatalf("%v %v", ids, err)
	}
}

func TestCircuitBreaker(t *testing.T) {
	now := time.Unix(0, 0)
	b := NewBreaker(2, time.Minute)
	b.now = func() time.Time { return now }
	var calls atomic.Int32
	failing := func() error { calls.Add(1); return errors.New("down") }

	b.Call(failing)
	b.Call(failing)
	if b.State() != "open" {
		t.Fatalf("state %s", b.State())
	}
	if err := b.Call(failing); !errors.Is(err, ErrOpen) || calls.Load() != 2 {
		t.Fatalf("open circuit still called the dependency (%v, %d calls)", err, calls.Load())
	}

	now = now.Add(time.Minute) // cooldown over: one trial call
	if err := b.Call(failing); err == nil || b.State() != "open" {
		t.Fatal("failed trial should re-open")
	}
	now = now.Add(time.Minute)
	if err := b.Call(func() error { return nil }); err != nil || b.State() != "closed" {
		t.Fatalf("successful trial should close: %v %s", err, b.State())
	}
}
