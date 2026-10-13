package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims are the token's payload. Anyone can READ them (base64), so never put secrets inside.
type Claims struct {
	Role Role   `json:"role"`
	Name string `json:"name"`
	jwt.RegisteredClaims
}

type Tokens struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
}

func NewTokens(secret string, ttl time.Duration) (*Tokens, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes")
	}
	return &Tokens{secret: []byte(secret), issuer: "go-tutorial", ttl: ttl, now: time.Now}, nil
}

func (t *Tokens) Issue(u User) (string, error) {
	now := t.now()
	claims := Claims{
		Role: u.Role,
		Name: u.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(u.ID, 10),
			Issuer:    t.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)), // short-lived: a stolen token expires soon
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

// Verify checks the signature, expiry, issuer AND the algorithm. Pinning the algorithm blocks the
// classic "alg: none" / algorithm-confusion attacks.
func (t *Tokens) Verify(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(t.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
