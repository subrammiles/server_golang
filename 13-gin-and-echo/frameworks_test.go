package frameworks

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/subrammiles/server_golang/12-rest-api/course"
)

func init() { gin.SetMode(gin.TestMode) }

// Both frameworks produce an http.Handler, so ONE test table checks both behave the same.
func TestBothFrameworks(t *testing.T) {
	svc := course.NewService(course.NewMemoryRepository(course.Seed()...))
	apps := map[string]http.Handler{"gin": NewGin(svc), "echo": NewEcho(svc)}

	tests := []struct {
		name, method, path, body string
		status                   int
		contains                 string
	}{
		{"ping", "GET", "/ping", "", 200, `"message":"pong"`},
		{"list users", "GET", "/users", "", 200, `"name":"subbu"`},
		{"create user", "POST", "/users", `{"name":"meera"}`, 201, `"name":"meera"`},
		{"validation", "POST", "/users", `{"name":"x"}`, 400, ""},
		{"html template", "GET", "/", "", 200, "Home Page"},
		{"course from lesson 12 service", "GET", "/api/courses/1", "", 200, "Go Fundamentals"},
		{"course not found", "GET", "/api/courses/99", "", 404, "not found"},
		{"bad id", "GET", "/api/courses/abc", "", 400, "number"},
	}
	for name, app := range apps {
		for _, tt := range tests {
			t.Run(name+"/"+tt.name, func(t *testing.T) {
				req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				app.ServeHTTP(rec, req)
				if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.contains) {
					t.Fatalf("got %d %s", rec.Code, rec.Body.String())
				}
			})
		}
	}
}
