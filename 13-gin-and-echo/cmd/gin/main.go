// go run ./13-gin-and-echo/cmd/gin     then: curl localhost:8080/ping
package main

import (
	frameworks "github.com/subrammiles/server_golang/13-gin-and-echo"

	"github.com/subrammiles/server_golang/12-rest-api/course"
)

func main() {
	svc := course.NewService(course.NewMemoryRepository(course.Seed()...))
	frameworks.NewGin(svc).Run(":8080") // listen and serve on 0.0.0.0:8080
}
