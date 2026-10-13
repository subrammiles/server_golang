// Package testing06 is code under test for the testing lesson.
package testing06

import (
	"errors"
	"strings"
	"unicode"
)

// Slugify turns a title into a URL-friendly slug: "Hello, World!" -> "hello-world".
func Slugify(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

var (
	ErrTooShort = errors.New("password too short")
	ErrNoDigit  = errors.New("password needs a digit")
)

func ValidatePassword(p string) error {
	if len(p) < 8 {
		return ErrTooShort
	}
	if !strings.ContainsFunc(p, unicode.IsDigit) {
		return ErrNoDigit
	}
	return nil
}

// Notifier is a DEPENDENCY behind an interface, so tests can pass a fake.
type Notifier interface {
	Notify(user, message string) error
}

type SignupService struct {
	Notifier Notifier
}

func (s SignupService) Register(user, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	return s.Notifier.Notify(user, "Welcome, "+user+"!")
}
