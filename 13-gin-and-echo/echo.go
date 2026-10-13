package frameworks

import (
	"errors"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/subrammiles/server_golang/12-rest-api/course"
)

// NewEcho builds the same API in Echo. Echo handlers RETURN an error, and a central handler renders it.
func NewEcho(svc *course.Service) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Renderer = &templateRenderer{template.Must(template.ParseGlob(templatesGlob()))}

	e.GET("/ping", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "pong"})
	})

	users := []User{{1, "subbu"}, {2, "ravi"}}
	e.GET("/users", func(c echo.Context) error { return c.JSON(http.StatusOK, users) })
	e.POST("/users", func(c echo.Context) error {
		var u User
		if err := c.Bind(&u); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON")
		}
		if len(u.Name) < 2 { // Echo has no built-in validator: plug one in or check by hand
			return echo.NewHTTPError(http.StatusBadRequest, "name must have at least 2 characters")
		}
		u.ID = len(users) + 1
		users = append(users, u)
		return c.JSON(http.StatusCreated, u)
	})

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "home.html", map[string]any{"title": "Home Page (Echo)"})
	})

	api := e.Group("/api")
	api.GET("/courses/:id", func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "id must be a number")
		}
		found, err := svc.Get(c.Request().Context(), id)
		if errors.Is(err, course.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		if err != nil {
			return err // becomes a 500 via Echo's error handler
		}
		return c.JSON(http.StatusOK, found)
	})
	return e
}

type templateRenderer struct{ t *template.Template }

func (r *templateRenderer) Render(w io.Writer, name string, data any, _ echo.Context) error {
	return r.t.ExecuteTemplate(w, name, data)
}

// templatesGlob finds templates/ next to this source file, so tests and `go run` work from any directory.
func templatesGlob() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "templates", "*.html")
}
