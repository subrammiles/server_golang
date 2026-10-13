// JWT_SECRET=$(openssl rand -base64 48) go run ./16-auth-security/cmd/server
package main

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	auth "github.com/subrammiles/server_golang/16-auth-security"
)

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		b := make([]byte, 48)
		rand.Read(b)
		secret = base64.StdEncoding.EncodeToString(b)
		slog.Warn("JWT_SECRET not set: using a random secret (tokens die on restart)")
	}
	tokens, err := auth.NewTokens(secret, 15*time.Minute)
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}
	users := auth.NewUserStore(bcrypt.DefaultCost)
	users.Register("admin@example.com", "admin-password-123", "Admin", auth.RoleAdmin)

	slog.Info("listening on :8080 (demo admin: admin@example.com / admin-password-123)")
	srv := &http.Server{Addr: ":8080", ReadHeaderTimeout: 5 * time.Second, Handler: auth.NewRouter(auth.Config{
		Users: users, Tokens: tokens, AllowedOrigins: []string{"http://localhost:5173"},
		LoginEvery: 10 * time.Second, LoginBurst: 5,
	})}
	slog.Error("server", "error", srv.ListenAndServe())
}
