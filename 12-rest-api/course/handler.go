package course

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
)

// Handler is the HTTP adapter: decode request -> call service -> encode response. No business rules here.
type Handler struct {
	svc *Service
	log *slog.Logger
}

func NewHandler(svc *Service, log *slog.Logger) *Handler { return &Handler{svc: svc, log: log} }

// Register adds the routes to a mux (Go 1.22 patterns).
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/courses", h.list)
	mux.HandleFunc("POST /api/courses", h.create)
	mux.HandleFunc("GET /api/courses/{id}", h.get)
	mux.HandleFunc("PUT /api/courses/{id}", h.replace)
	mux.HandleFunc("PATCH /api/courses/{id}", h.patch)
	mux.HandleFunc("DELETE /api/courses/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, err1 := intParam(q.Get("page"), 0)
	size, err2 := intParam(q.Get("size"), 10)
	if err := errors.Join(err1, err2); err != nil {
		h.problem(w, r, &ValidationError{Fields: map[string]string{"page/size": "must be integers"}})
		return
	}
	sort := q.Get("sort")
	if sort == "" {
		sort = "title"
	}
	result, err := h.svc.List(r.Context(), Query{Category: q.Get("category"), Q: q.Get("q"), Page: page, Size: size, Sort: sort})
	if err != nil {
		h.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	c, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !h.decode(w, r, &in) {
		return
	}
	c, err := h.svc.Create(r.Context(), in)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/courses/"+strconv.FormatInt(c.ID, 10))
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) replace(w http.ResponseWriter, r *http.Request) {
	id, ok := h.id(w, r)
	var in Input
	if !ok || !h.decode(w, r, &in) {
		return
	}
	c, err := h.svc.Replace(r.Context(), id, in)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id, ok := h.id(w, r)
	var p Patch
	if !ok || !h.decode(w, r, &p) {
		return
	}
	c, err := h.svc.Update(r.Context(), id, p)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- helpers ----

func (h *Handler) id(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		h.problem(w, r, &ValidationError{Fields: map[string]string{"id": "must be a positive integer"}})
		return 0, false
	}
	return id, true
}

func (h *Handler) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid JSON body: "+err.Error(), nil)
		return false
	}
	return true
}

// problem maps domain errors to HTTP status codes in ONE place (RFC 9457 problem details).
func (h *Handler) problem(w http.ResponseWriter, r *http.Request, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		writeProblem(w, r, http.StatusBadRequest, "Validation failed", "the request has invalid fields", ve.Fields)
	case errors.Is(err, ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "Not Found", err.Error(), nil)
	case errors.Is(err, ErrDuplicate):
		writeProblem(w, r, http.StatusConflict, "Conflict", err.Error(), nil)
	default:
		h.log.Error("unexpected error", "error", err, "path", r.URL.Path) // log details, return nothing internal
		writeProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "something went wrong", nil)
	}
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, title, detail string, fields map[string]string) {
	body := map[string]any{"type": "about:blank", "title": title, "status": status, "detail": detail, "instance": r.URL.Path}
	if fields != nil {
		body["errors"] = fields
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func intParam(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	return strconv.Atoi(s)
}
