// Package errorhandling covers Go's approach: errors are ordinary VALUES returned from functions.
package errorhandling

import (
	"errors"
	"fmt"
	"strconv"
)

// SENTINEL errors: package-level values callers can compare against with errors.Is.
var (
	ErrNotFound     = errors.New("not found")
	ErrInsufficient = errors.New("insufficient funds")
)

// ValidationError is a CUSTOM error type carrying extra data. Callers extract it with errors.As.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Reason)
}

type Account struct {
	ID      string
	Balance int
}

type Bank struct {
	accounts map[string]*Account
}

func NewBank(accounts ...*Account) *Bank {
	b := &Bank{accounts: map[string]*Account{}}
	for _, a := range accounts {
		b.accounts[a.ID] = a
	}
	return b
}

func (b *Bank) find(id string) (*Account, error) {
	acc, ok := b.accounts[id]
	if !ok {
		// WRAP with %w: adds context while keeping the original error inspectable
		return nil, fmt.Errorf("account %q: %w", id, ErrNotFound)
	}
	return acc, nil
}

// Transfer shows the everyday pattern: check every error, add context, return early.
func (b *Bank) Transfer(from, to string, amount int) error {
	if amount <= 0 {
		return &ValidationError{Field: "amount", Reason: "must be positive"}
	}
	src, err := b.find(from)
	if err != nil {
		return fmt.Errorf("transfer failed: %w", err)
	}
	dst, err := b.find(to)
	if err != nil {
		return fmt.Errorf("transfer failed: %w", err)
	}
	if src.Balance < amount {
		return fmt.Errorf("transfer of %d from %s: %w", amount, from, ErrInsufficient)
	}
	src.Balance -= amount
	dst.Balance += amount
	return nil
}

// ParsePort combines a standard library error with our own checks.
func ParsePort(s string) (int, error) {
	port, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse port: %w", err)
	}
	if port < 1 || port > 65535 {
		return 0, &ValidationError{Field: "port", Reason: "must be 1-65535"}
	}
	return port, nil
}

// SafeDivide shows PANIC and RECOVER. Panics are for programmer bugs (nil pointer, index out of range),
// NOT for expected failures. recover() inside a deferred function turns a panic back into an error,
// e.g. so one bad HTTP request doesn't crash the whole server.
func SafeDivide(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered: %v", r)
		}
	}()
	return a / b, nil // integer division by zero panics
}

// CloseAll uses errors.Join to report several failures at once.
func CloseAll(closers ...func() error) error {
	var errs []error
	for _, c := range closers {
		if err := c(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...) // nil if errs is empty
}
