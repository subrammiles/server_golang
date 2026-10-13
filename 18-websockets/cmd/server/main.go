// go run ./18-websockets/cmd/server     then open http://localhost:8080 in two browser tabs
package main

import (
	"log/slog"
	"net/http"
	"time"

	chat "github.com/subrammiles/server_golang/18-websockets"
)

func main() {
	slog.Info("chat on http://localhost:8080")
	srv := &http.Server{Addr: ":8080", Handler: chat.NewHub(30 * time.Second).Handler(), ReadHeaderTimeout: 5 * time.Second}
	slog.Error("server", "error", srv.ListenAndServe())
}
