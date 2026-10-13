package auth

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type userKey struct{}

// Authenticate requires "Authorization: Bearer <jwt>" and puts the caller's claims in the context (401 otherwise).
func Authenticate(tokens *Tokens) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok {
				w.Header().Set("WWW-Authenticate", "Bearer")
				jsonError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			claims, err := tokens.Verify(raw)
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
				jsonError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, claims)))
		})
	}
}

func CurrentUser(ctx context.Context) (id int64, claims *Claims) {
	claims, _ = ctx.Value(userKey{}).(*Claims)
	if claims != nil {
		id, _ = strconv.ParseInt(claims.Subject, 10, 64)
	}
	return id, claims
}

// RequireRole is AUTHORIZATION: authenticated but not allowed means 403.
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, c := CurrentUser(r.Context()); c == nil || !slices.Contains(roles, c.Role) {
				jsonError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders: cheap browser protections every API/site should send.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")                                     // no MIME sniffing
		h.Set("X-Frame-Options", "DENY")                                               // no clickjacking via iframes
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'") // JSON API: load nothing
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains") // browsers: HTTPS only
		next.ServeHTTP(w, r)
	})
}

// CORS allows listed BROWSER origins to call the API. CORS is not authentication: curl ignores it.
func CORS(allowed ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && slices.Contains(allowed, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE")
			}
			if r.Method == http.MethodOptions { // preflight request
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit allows `burst` requests per client IP, refilling at `every`. Brakes password guessing.
// (Behind a load balancer use the real client IP header; with several instances use a shared store such as Redis.)
func RateLimit(every time.Duration, burst int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	limiters := map[string]*rate.Limiter{}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			mu.Lock()
			l, ok := limiters[ip]
			if !ok {
				l = rate.NewLimiter(rate.Every(every), burst) // token bucket
				limiters[ip] = l
			}
			mu.Unlock()
			if !l.Allow() {
				w.Header().Set("Retry-After", strconv.Itoa(int(every.Seconds())+1))
				jsonError(w, http.StatusTooManyRequests, "too many requests, slow down")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
