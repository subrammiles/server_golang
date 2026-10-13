package restapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/subrammiles/server_golang/12-rest-api/course"
	"github.com/subrammiles/server_golang/12-rest-api/course/coursetest"
)

func newAPI() http.Handler {
	return NewRouter(course.NewMemoryRepository(course.Seed()...), slog.New(slog.DiscardHandler))
}

func call(t *testing.T, h http.Handler, method, path, body string) (int, map[string]any, http.Header) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out, rec.Header()
}

func TestCRUDLifecycle(t *testing.T) {
	api := newAPI()
	code, created, hdr := call(t, api, "POST", "/api/courses", `{"title":"Docker","category":"devops","durationHours":6,"price":799}`)
	if code != 201 || hdr.Get("Location") != "/api/courses/4" {
		t.Fatalf("create: %d %v %v", code, created, hdr)
	}
	if code, c, _ := call(t, api, "PATCH", "/api/courses/4", `{"price":699}`); code != 200 || c["price"] != 699.0 || c["title"] != "Docker" {
		t.Fatalf("patch: %d %v", code, c)
	}
	if code, c, _ := call(t, api, "PUT", "/api/courses/4", `{"title":"Docker Deep Dive","category":"devops","durationHours":9,"price":999}`); code != 200 || c["durationHours"] != 9.0 {
		t.Fatalf("put: %d %v", code, c)
	}
	if code, _, _ := call(t, api, "DELETE", "/api/courses/4", ""); code != 204 {
		t.Fatalf("delete: %d", code)
	}
	if code, p, _ := call(t, api, "GET", "/api/courses/4", ""); code != 404 || p["detail"] != "course not found" {
		t.Fatalf("get after delete: %d %v", code, p)
	}
}

func TestListPagingFilteringSorting(t *testing.T) {
	code, page, _ := call(t, newAPI(), "GET", "/api/courses?category=backend&sort=-price&size=1", "")
	items := page["items"].([]any)
	if code != 200 || page["totalElements"] != 2.0 || page["totalPages"] != 2.0 || items[0].(map[string]any)["title"] != "Go Fundamentals" {
		t.Fatalf("%d %v", code, page)
	}
}

func TestErrorsAreProblemDetails(t *testing.T) {
	api := newAPI()
	tests := []struct {
		name, method, path, body string
		status                   int
	}{
		{"validation lists every field", "POST", "/api/courses", `{"title":"","category":"","durationHours":0,"price":-1}`, 400},
		{"unknown JSON field", "POST", "/api/courses", `{"title":"x","id":7}`, 400},
		{"bad query parameter", "GET", "/api/courses?size=1000", "", 400},
		{"bad sort", "GET", "/api/courses?sort=password", "", 400},
		{"bad id", "GET", "/api/courses/abc", "", 400},
		{"duplicate title is a conflict", "POST", "/api/courses", `{"title":"rest api design","category":"x","durationHours":1,"price":1}`, 409},
		{"not found", "DELETE", "/api/courses/999", "", 404},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			api.ServeHTTP(rec, req)
			if rec.Code != tt.status || rec.Header().Get("Content-Type") != "application/problem+json" {
				t.Fatalf("got %d %s %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body)
			}
		})
	}
	_, p, _ := call(t, api, "POST", "/api/courses", `{"title":"","category":"","durationHours":0,"price":-1}`)
	if len(p["errors"].(map[string]any)) != 4 {
		t.Fatalf("want 4 field errors, got %v", p["errors"])
	}
}

func TestHTMLPageEscapesData(t *testing.T) {
	repo := course.NewMemoryRepository(course.Input{Title: `<script>alert("xss")</script>`, Category: "x", DurationHours: 1, Price: 1})
	rec := httptest.NewRecorder()
	NewRouter(repo, slog.New(slog.DiscardHandler)).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	body := rec.Body.String()
	if rec.Code != 200 || strings.Contains(body, "<script>alert") || !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("not escaped: %s", body)
	}
}

func TestMemoryRepositoryContract(t *testing.T) {
	coursetest.RunContract(t, func(t *testing.T) course.Repository { return course.NewMemoryRepository(course.Seed()...) })
}
