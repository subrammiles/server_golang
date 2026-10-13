// Package httpserver builds an HTTP API with ONLY the standard library. Since Go 1.22, net/http's router
// supports methods and path wildcards, so many Go services need no web framework at all.
package httpserver

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed static
var staticFiles embed.FS

type Book struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type store struct {
	mu     sync.Mutex // handlers run CONCURRENTLY (one goroutine per request): protect shared state
	books  map[int]Book
	nextID int
}

// NewHandler wires routes and middleware. Returning an http.Handler (not starting a server) makes it testable.
func NewHandler(log *slog.Logger) http.Handler {
	s := &store{books: map[int]Book{1: {1, "The Go Programming Language"}}, nextID: 2}
	mux := http.NewServeMux()

	// Go 1.22 patterns: "METHOD /path/{wildcard}". Wrong method -> automatic 405; unknown path -> 404.
	mux.HandleFunc("GET /hello/{name}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"message": "Hello, " + r.PathValue("name") + "!"})
	})
	mux.HandleFunc("GET /books", s.list)
	mux.HandleFunc("POST /books", s.create)
	mux.HandleFunc("GET /books/{id}", s.get)
	mux.HandleFunc("DELETE /books/{id}", s.delete)
	mux.HandleFunc("GET /slow", slow)
	mux.HandleFunc("GET /panic", func(http.ResponseWriter, *http.Request) { panic("something went very wrong") })

	static, _ := fs.Sub(staticFiles, "static")
	mux.Handle("GET /", http.FileServerFS(static)) // "/" matches everything not matched above

	return Chain(mux, RequestID, Logger(log), Recover(log))
}

func (s *store) list(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q")) // query string: /books?q=go
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Book{} // not nil, so JSON is [] instead of null
	for _, b := range s.books {
		if strings.Contains(strings.ToLower(b.Title), q) {
			out = append(out, b)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *store) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title string `json:"title"`
	}
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, err)
		return
	}
	if strings.TrimSpace(in.Title) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}
	s.mu.Lock()
	b := Book{ID: s.nextID, Title: strings.TrimSpace(in.Title)}
	s.books[b.ID] = b
	s.nextID++
	s.mu.Unlock()
	w.Header().Set("Location", "/books/"+strconv.Itoa(b.ID))
	writeJSON(w, http.StatusCreated, b)
}

func (s *store) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id must be a number"})
		return
	}
	s.mu.Lock()
	b, ok := s.books[id]
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "book not found"})
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *store) delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	s.mu.Lock()
	_, ok := s.books[id]
	delete(s.books, id)
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "book not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// slow honours the request CONTEXT: if the client disconnects (or the server shuts down), stop working.
func slow(w http.ResponseWriter, r *http.Request) {
	d, _ := time.ParseDuration(r.URL.Query().Get("d"))
	select {
	case <-time.After(d):
		writeJSON(w, http.StatusOK, map[string]string{"status": "done"})
	case <-r.Context().Done():
		return
	}
}

// ---- JSON helpers ----

type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

// readJSON decodes a body safely: size limit, Content-Type check, no unknown fields, one JSON value only.
func readJSON(w http.ResponseWriter, r *http.Request, v any) error {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		return &httpError{http.StatusUnsupportedMediaType, "Content-Type must be application/json"}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return &httpError{http.StatusRequestEntityTooLarge, "body too large"}
		}
		return &httpError{http.StatusBadRequest, "invalid JSON: " + err.Error()}
	}
	return nil
}

func writeError(w http.ResponseWriter, err error) {
	var he *httpError
	if errors.As(err, &he) {
		writeJSON(w, he.status, map[string]string{"error": he.msg})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
