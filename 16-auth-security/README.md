# 16: Authentication and security

```
POST /auth/register {email, password, name}   -> 201, password stored as a bcrypt hash, role "user"
POST /auth/login    {email, password}         -> {accessToken: "eyJ..."} (JWT, 15 min)
GET  /api/me        Authorization: Bearer eyJ...
```

## Concepts
- **Authentication** (who? failure is **401**) vs **authorization** (allowed? failure is **403**).
- **Password storage**: `bcrypt` (`golang.org/x/crypto`), salted and slow. The hash is an **unexported** field, so it can never reach JSON.
  bcrypt only uses the first 72 bytes, hence the length limit.
- **JWT** (`golang-jwt/jwt/v5`): signed, short-lived claims. `Verify` pins the **algorithm** (blocks `alg: none`),
  and checks issuer and expiry. The secret comes from `JWT_SECRET`, and short secrets are refused.
- **Middleware**: `Authenticate` (claims in the context), `RequireRole`, and **ownership** checks against IDOR.
- **Mass assignment**: `DisallowUnknownFields` stops `{"role":"admin"}` at registration.
- **No user enumeration**: the same error and the same bcrypt work for unknown emails and wrong passwords.
- **Rate limiting** with a token bucket (`x/time/rate`) per IP gives 429 + `Retry-After`. Use Redis with several instances.
- **Security headers** (nosniff, frame denial, CSP, HSTS) and a **CORS** allowlist with preflight. CORS isn't authentication.
- Also know: TLS everywhere, refresh tokens and revocation, cookie sessions (`HttpOnly`, `Secure`, `SameSite`) plus CSRF,
  OAuth2/OIDC (`golang.org/x/oauth2`), `govulncheck ./...` for vulnerable dependencies, and the OWASP Top 10.

## Run
```bash
JWT_SECRET=$(openssl rand -base64 48) go run ./16-auth-security/cmd/server
TOKEN=$(curl -s -X POST localhost:8080/auth/login -d '{"email":"admin@example.com","password":"admin-password-123"}' | jq -r .accessToken)
curl -H "Authorization: Bearer $TOKEN" localhost:8080/api/me
go test -v ./16-auth-security
```
