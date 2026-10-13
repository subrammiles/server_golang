package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"time"
)

type Config struct {
	Users          *UserStore
	Tokens         *Tokens
	AllowedOrigins []string
	LoginEvery     time.Duration // rate limit refill interval for /auth/login
	LoginBurst     int
}

func NewRouter(cfg Config) http.Handler {
	mux := http.NewServeMux()
	authn := Authenticate(cfg.Tokens)
	admin := RequireRole(RoleAdmin)

	mux.HandleFunc("POST /auth/register", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Email, Password, Name string
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		dec.DisallowUnknownFields() // {"role":"admin"} is rejected: clients can't pick their own role
		if err := dec.Decode(&in); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if _, err := mail.ParseAddress(in.Email); err != nil || len(in.Password) < 10 || len(in.Password) > 72 || in.Name == "" {
			jsonError(w, http.StatusBadRequest, "need a valid email, a name, and a password of 10-72 characters")
			return
		}
		u, err := cfg.Users.Register(in.Email, in.Password, in.Name, RoleUser)
		if errors.Is(err, ErrEmailTaken) {
			jsonError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, u)
	})

	login := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Email, Password string }
		json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in)
		u, err := cfg.Users.Authenticate(in.Email, in.Password)
		if err != nil {
			jsonError(w, http.StatusUnauthorized, ErrInvalidCredentials.Error())
			return
		}
		token, _ := cfg.Tokens.Issue(u)
		writeJSON(w, http.StatusOK, map[string]any{"accessToken": token, "tokenType": "Bearer", "expiresIn": int(cfg.Tokens.ttl.Seconds())})
	})
	mux.Handle("POST /auth/login", RateLimit(cfg.LoginEvery, cfg.LoginBurst)(login))

	courses := []string{"Go Security Basics"}
	mux.HandleFunc("GET /api/courses", func(w http.ResponseWriter, r *http.Request) { // public
		writeJSON(w, http.StatusOK, courses)
	})
	mux.Handle("POST /api/courses", authn(admin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		courses = append(courses, "New course")
		writeJSON(w, http.StatusCreated, courses)
	}))))

	mux.Handle("GET /api/me", authn(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := CurrentUser(r.Context())
		u, _ := cfg.Users.ByID(id)
		writeJSON(w, http.StatusOK, u)
	})))

	// OWNERSHIP: users see only their own orders, admins anyone's (prevents IDOR: guessing other ids)
	mux.Handle("GET /api/users/{id}/orders", authn(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		me, claims := CurrentUser(r.Context())
		if me != target && claims.Role != RoleAdmin {
			jsonError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeJSON(w, http.StatusOK, []map[string]int64{{"orderId": 100 + target, "userId": target}})
	})))

	return SecurityHeaders(CORS(cfg.AllowedOrigins...)(mux))
}
