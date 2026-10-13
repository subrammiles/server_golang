// Package auth shows authentication (who are you?) and authorization (may you?) for an HTTP API.
package auth

import (
	"errors"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	Role         Role   `json:"role"`
	passwordHash []byte // unexported: can never be serialised to JSON by accident
}

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// UserStore keeps users in memory; passwords are stored as BCRYPT hashes (salted, deliberately slow).
type UserStore struct {
	mu     sync.Mutex
	byMail map[string]*User
	nextID int64
	cost   int
	// a hash of a random password, compared against when the email doesn't exist (see Authenticate)
	dummyHash []byte
}

// NewUserStore takes the bcrypt COST (work factor): bcrypt.DefaultCost (10) in production, higher as
// hardware gets faster. Tests pass bcrypt.MinCost so they stay fast; the slowness is the security.
func NewUserStore(cost int) *UserStore {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), cost)
	return &UserStore{byMail: map[string]*User{}, nextID: 1, cost: cost, dummyHash: dummy}
}

func (s *UserStore) Register(email, password, name string, role Role) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost) // cost 10: ~50-100 ms per hash
	if err != nil {
		return User{}, err // e.g. password longer than 72 bytes
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.ToLower(email)
	if _, exists := s.byMail[key]; exists {
		return User{}, ErrEmailTaken
	}
	u := &User{ID: s.nextID, Email: key, Name: name, Role: role, passwordHash: hash}
	s.byMail[key] = u
	s.nextID++
	return *u, nil
}

// Authenticate returns the SAME error for "no such email" and "wrong password", and spends the same
// time in both cases, so attackers can't discover which emails are registered (user enumeration).
func (s *UserStore) Authenticate(email, password string) (User, error) {
	s.mu.Lock()
	u, ok := s.byMail[strings.ToLower(email)]
	s.mu.Unlock()
	if !ok {
		bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return User{}, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword(u.passwordHash, []byte(password)) != nil {
		return User{}, ErrInvalidCredentials
	}
	return *u, nil
}

func (s *UserStore) ByID(id int64) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.byMail {
		if u.ID == id {
			return *u, true
		}
	}
	return User{}, false
}

// HashOf is for tests only: proves we store a hash, not the password.
func (s *UserStore) HashOf(email string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.byMail[strings.ToLower(email)].passwordHash
}
