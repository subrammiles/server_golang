// Package frameworks shows the same small API in Gin and Echo (the two frameworks from the old version
// of this repo), plus lesson 12's course service, so you can compare them with plain net/http.
package frameworks

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/subrammiles/server_golang/12-rest-api/course"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name" binding:"required,min=2"` // `binding` = validation rules (go-playground/validator)
}

// NewGin builds the Gin router. gin.New() + explicit middleware; gin.Default() would add Logger and Recovery itself.
func NewGin(svc *course.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.LoadHTMLGlob(templatesGlob())

	// From the old 1_simple_get.go
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"}) // gin.H = map[string]any
	})

	// From the old 2_get_with_struct.go, plus binding/validation on POST
	users := []User{{1, "subbu"}, {2, "ravi"}}
	r.GET("/users", func(c *gin.Context) { c.JSON(http.StatusOK, users) })
	r.POST("/users", func(c *gin.Context) {
		var u User
		if err := c.ShouldBindJSON(&u); err != nil { // decode + validate in one call
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		u.ID = len(users) + 1
		users = append(users, u)
		c.JSON(http.StatusCreated, u)
	})

	// From the old 3_inline_html_page.go and 4_external_html_page.go
	r.GET("/inline", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html><body>Hello</body></html>"))
	})
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home.html", gin.H{"title": "Home Page (Gin)"})
	})

	// Route groups share a prefix and middleware
	api := r.Group("/api")
	api.GET("/courses/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id must be a number"})
			return
		}
		found, err := svc.Get(c.Request.Context(), id)
		if errors.Is(err, course.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, found)
	})
	return r
}
