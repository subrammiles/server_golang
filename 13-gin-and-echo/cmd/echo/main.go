// go run ./13-gin-and-echo/cmd/echo    then: curl localhost:1323/ping
package main

import (
	frameworks "github.com/subrammiles/server_golang/13-gin-and-echo"

	"github.com/subrammiles/server_golang/12-rest-api/course"
)

func main() {
	svc := course.NewService(course.NewMemoryRepository(course.Seed()...))
	e := frameworks.NewEcho(svc)
	e.Logger.Fatal(e.Start(":1323"))
}
