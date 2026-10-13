// Package course is the domain + business logic of the course catalog, independent of HTTP and storage.
package course

import (
	"context"
	"errors"
	"strings"
	"time"
)

type Course struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	Category      string    `json:"category"`
	DurationHours int       `json:"durationHours"`
	Price         int64     `json:"price"` // whole rupees: integers avoid floating-point money bugs
	CreatedAt     time.Time `json:"createdAt"`
}

// Input is what clients may send (no id, no createdAt).
type Input struct {
	Title         string `json:"title"`
	Category      string `json:"category"`
	DurationHours int    `json:"durationHours"`
	Price         int64  `json:"price"`
}

// Patch: pointer fields distinguish "not sent" (nil) from "set to the zero value".
type Patch struct {
	Title         *string `json:"title"`
	Category      *string `json:"category"`
	DurationHours *int    `json:"durationHours"`
	Price         *int64  `json:"price"`
}

type Query struct {
	Category, Q string
	Page, Size  int
	Sort        string // title, -title, price, -price, durationHours, -durationHours
}

var (
	ErrNotFound  = errors.New("course not found")
	ErrDuplicate = errors.New("a course with this title already exists")
)

// ValidationError lists every invalid field at once.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "validation failed" }

// Repository is the storage PORT. In-memory, SQLite (lesson 14) and PostgreSQL (lesson 15) implement it;
// coursetest.RunContract checks that they all behave the same.
type Repository interface {
	List(ctx context.Context, q Query) (items []Course, total int, err error)
	Get(ctx context.Context, id int64) (Course, error)
	GetByTitle(ctx context.Context, title string) (Course, error) // case-insensitive
	Create(ctx context.Context, in Input) (Course, error)
	Update(ctx context.Context, c Course) (Course, error)
	Delete(ctx context.Context, id int64) error
}

var validSorts = map[string]bool{"title": true, "-title": true, "price": true, "-price": true, "durationHours": true, "-durationHours": true}

func (in Input) normalize() Input {
	in.Title = strings.TrimSpace(in.Title)
	in.Category = strings.TrimSpace(in.Category)
	return in
}

func (in Input) validate() error {
	errs := map[string]string{}
	if in.Title == "" || len(in.Title) > 100 {
		errs["title"] = "required, at most 100 characters"
	}
	if in.Category == "" {
		errs["category"] = "required"
	}
	if in.DurationHours <= 0 || in.DurationHours > 500 {
		errs["durationHours"] = "must be between 1 and 500"
	}
	if in.Price < 0 {
		errs["price"] = "must not be negative"
	}
	if len(errs) > 0 {
		return &ValidationError{Fields: errs}
	}
	return nil
}

func (q Query) validate() error {
	errs := map[string]string{}
	if q.Page < 0 {
		errs["page"] = "must be >= 0"
	}
	if q.Size < 1 || q.Size > 100 {
		errs["size"] = "must be between 1 and 100"
	}
	if !validSorts[q.Sort] {
		errs["sort"] = "unknown sort field"
	}
	if len(errs) > 0 {
		return &ValidationError{Fields: errs}
	}
	return nil
}

// Seed is the sample data every repository starts with (shared with the contract tests).
func Seed() []Input {
	return []Input{
		{Title: "Go Fundamentals", Category: "backend", DurationHours: 12, Price: 1999},
		{Title: "Data Structures and Algorithms", Category: "cs", DurationHours: 40, Price: 2999},
		{Title: "REST API Design", Category: "backend", DurationHours: 8, Price: 999},
	}
}
