// Package web renders server-side HTML with html/template (ported from the old gin template example).
package web

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/subrammiles/server_golang/12-rest-api/course"
)

//go:embed templates/*.html
var files embed.FS

// Parse templates ONCE at startup (fails fast on syntax errors), not on every request.
var pages = template.Must(template.ParseFS(files, "templates/*.html"))

// CoursesPage lists courses as HTML. html/template ESCAPES data automatically, which prevents XSS:
// a course titled "<script>..." is shown as text, never executed.
func CoursesPage(svc *course.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := svc.List(r.Context(), course.Query{Page: 0, Size: 100, Sort: "title"})
		if err != nil {
			http.Error(w, "could not load courses", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		pages.ExecuteTemplate(w, "courses.html", map[string]any{"Title": "Course catalog", "Courses": page.Items})
	}
}
