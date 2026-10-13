package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type testAPI struct {
	t      *testing.T
	h      http.Handler
	users  *UserStore
	tokens *Tokens
}

func newTestAPI(t *testing.T) *testAPI {
	users := NewUserStore(bcrypt.MinCost) // fast hashing for tests
	users.Register("admin@example.com", "admin-password-123", "Admin", RoleAdmin)
	tokens, err := NewTokens("test-secret-that-is-long-enough-1234567890", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	h := NewRouter(Config{Users: users, Tokens: tokens, AllowedOrigins: []string{"http://localhost:5173"},
		LoginEvery: time.Hour, LoginBurst: 3})
	return &testAPI{t, h, users, tokens}
}

func (a *testAPI) do(method, path, body, token string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	a.h.ServeHTTP(rec, req)
	return rec
}

func (a *testAPI) login(email, password string) string {
	a.t.Helper()
	rec := a.do("POST", "/auth/login", `{"email":"`+email+`","password":"`+password+`"}`, "")
	if rec.Code != 200 {
		a.t.Fatalf("login %s: %d %s", email, rec.Code, rec.Body)
	}
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return out["accessToken"].(string)
}

func TestRegisterLoginAndProtectedEndpoint(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do("POST", "/auth/register", `{"email":"Meera@Example.com","password":"correct-horse-battery","name":"Meera"}`, "")
	if rec.Code != 201 || !strings.Contains(rec.Body.String(), `"role":"user"`) || strings.Contains(rec.Body.String(), "hash") {
		t.Fatalf("register: %d %s", rec.Code, rec.Body)
	}
	token := a.login("meera@example.com", "correct-horse-battery")
	if rec := a.do("GET", "/api/me", "", token); rec.Code != 200 || !strings.Contains(rec.Body.String(), "meera@example.com") {
		t.Fatalf("me: %d %s", rec.Code, rec.Body)
	}
	if !bytes.HasPrefix(a.users.HashOf("meera@example.com"), []byte("$2a$")) {
		t.Fatal("password is not stored as a bcrypt hash")
	}
}

func TestRegistrationRules(t *testing.T) {
	a := newTestAPI(t)
	cases := map[string]struct {
		body   string
		status int
	}{
		"cannot choose own role": {`{"email":"x@example.com","password":"long-enough-pw","name":"X","role":"admin"}`, 400},
		"weak password":          {`{"email":"x@example.com","password":"short","name":"X"}`, 400},
		"bad email":              {`{"email":"not-an-email","password":"long-enough-pw","name":"X"}`, 400},
		"duplicate":              {`{"email":"ADMIN@example.com","password":"long-enough-pw","name":"X"}`, 409},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if rec := a.do("POST", "/auth/register", c.body, ""); rec.Code != c.status {
				t.Fatalf("got %d %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestNoUserEnumeration(t *testing.T) {
	a := newTestAPI(t)
	unknown := a.do("POST", "/auth/login", `{"email":"nobody@example.com","password":"x"}`, "")
	wrong := a.do("POST", "/auth/login", `{"email":"admin@example.com","password":"wrong"}`, "")
	if unknown.Code != 401 || unknown.Body.String() != wrong.Body.String() {
		t.Fatalf("responses differ: %s vs %s", unknown.Body, wrong.Body)
	}
}

func TestBadTokensAre401(t *testing.T) {
	a := newTestAPI(t)
	token := a.login("admin@example.com", "admin-password-123")

	parts := strings.Split(token, ".")
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	forged := strings.Replace(string(payload), `"sub":"1"`, `"sub":"2"`, 1)
	tampered := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(forged)) + "." + parts[2]

	expiredIssuer, _ := NewTokens("test-secret-that-is-long-enough-1234567890", time.Minute)
	expiredIssuer.now = func() time.Time { return time.Now().Add(-time.Hour) }
	expired, _ := expiredIssuer.Issue(User{ID: 1, Role: RoleAdmin})

	other, _ := NewTokens("a-completely-different-secret-0123456789", time.Minute)
	foreign, _ := other.Issue(User{ID: 1, Role: RoleAdmin})

	noneAlg := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`)) + "." + parts[1] + "."

	for name, tok := range map[string]string{"missing": "", "tampered": tampered, "expired": expired, "foreign key": foreign, "alg none": noneAlg} {
		t.Run(name, func(t *testing.T) {
			if rec := a.do("GET", "/api/me", "", tok); rec.Code != 401 || rec.Header().Get("WWW-Authenticate") == "" {
				t.Fatalf("got %d", rec.Code)
			}
		})
	}
}

func TestRolesAndOwnership(t *testing.T) {
	a := newTestAPI(t)
	a.users.Register("u@example.com", "user-password-1", "U", RoleUser)
	user := a.login("u@example.com", "user-password-1")
	admin := a.login("admin@example.com", "admin-password-123")

	checks := []struct {
		name, method, path, token string
		status                    int
	}{
		{"public", "GET", "/api/courses", "", 200},
		{"user cannot create", "POST", "/api/courses", user, 403},
		{"admin can create", "POST", "/api/courses", admin, 201},
		{"own orders", "GET", "/api/users/2/orders", user, 200},
		{"someone else's orders (IDOR)", "GET", "/api/users/1/orders", user, 403},
		{"admin sees anyone's", "GET", "/api/users/2/orders", admin, 200},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if rec := a.do(c.method, c.path, "", c.token); rec.Code != c.status {
				t.Fatalf("got %d", rec.Code)
			}
		})
	}
}

func TestLoginRateLimit(t *testing.T) {
	a := newTestAPI(t)
	for range 3 {
		a.do("POST", "/auth/login", `{"email":"admin@example.com","password":"guess"}`, "")
	}
	rec := a.do("POST", "/auth/login", `{"email":"admin@example.com","password":"admin-password-123"}`, "")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestSecurityHeadersAndCORS(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do("GET", "/api/courses", "", "", "Origin", "http://localhost:5173")
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("security headers missing")
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatal("allowed origin not echoed")
	}
	if a.do("GET", "/api/courses", "", "", "Origin", "https://evil.example").Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("foreign origin allowed")
	}
	if pre := a.do("OPTIONS", "/api/courses", "", "", "Origin", "http://localhost:5173"); pre.Code != 204 {
		t.Fatalf("preflight: %d", pre.Code)
	}
}

func TestWeakSecretRejected(t *testing.T) {
	if _, err := NewTokens("short", time.Minute); err == nil {
		t.Fatal("want an error")
	}
}
