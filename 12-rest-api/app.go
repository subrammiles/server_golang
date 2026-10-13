// Package restapi wires the course catalog together. Lessons 14 and 15 call NewRouter with a database repository.
package restapi

import (
	"log/slog"
	"net/http"

	"github.com/subrammiles/server_golang/12-rest-api/course"
	"github.com/subrammiles/server_golang/12-rest-api/web"
)

// NewRouter is the COMPOSITION ROOT: the one place that decides which implementations are used.
func NewRouter(repo course.Repository, log *slog.Logger) http.Handler {
	svc := course.NewService(repo)
	mux := http.NewServeMux()
	course.NewHandler(svc, log).Register(mux)
	mux.Handle("GET /{$}", web.CoursesPage(svc)) // {$} = exactly "/"
	return mux
}
