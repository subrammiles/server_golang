// Package coursetest holds a CONTRACT TEST that every course.Repository implementation must pass
// (in-memory here, SQLite in lesson 14, PostgreSQL in lesson 15). That's what makes them interchangeable.
package coursetest

import (
	"context"
	"errors"
	"testing"

	"github.com/subrammiles/server_golang/12-rest-api/course"
)

// RunContract runs the suite. newRepo must return a repository containing exactly course.Seed().
func RunContract(t *testing.T, newRepo func(t *testing.T) course.Repository) {
	ctx := context.Background()
	base := course.Query{Page: 0, Size: 10, Sort: "title"}

	t.Run("lists the seed sorted by title", func(t *testing.T) {
		items, total, err := newRepo(t).List(ctx, base)
		if err != nil || total != 3 {
			t.Fatalf("total %d, err %v", total, err)
		}
		want := []string{"Data Structures and Algorithms", "Go Fundamentals", "REST API Design"}
		for i, c := range items {
			if c.Title != want[i] {
				t.Errorf("item %d = %q, want %q", i, c.Title, want[i])
			}
		}
	})

	t.Run("filters, searches, sorts descending and pages", func(t *testing.T) {
		q := base
		q.Category, q.Q, q.Sort, q.Size = "backend", "e", "-price", 1
		items, total, err := newRepo(t).List(ctx, q)
		if err != nil || total != 2 || len(items) != 1 || items[0].Title != "Go Fundamentals" {
			t.Fatalf("got %+v total %d err %v", items, total, err)
		}
	})

	t.Run("create, get, get by title (case-insensitive)", func(t *testing.T) {
		repo := newRepo(t)
		created, err := repo.Create(ctx, course.Input{Title: "Kubernetes", Category: "devops", DurationHours: 20, Price: 2499})
		if err != nil || created.ID <= 3 || created.CreatedAt.IsZero() {
			t.Fatalf("created %+v err %v", created, err)
		}
		got, err := repo.Get(ctx, created.ID)
		if err != nil || got != created {
			t.Fatalf("get = %+v, want %+v (err %v)", got, created, err)
		}
		byTitle, err := repo.GetByTitle(ctx, "KUBERNETES")
		if err != nil || byTitle.ID != created.ID {
			t.Fatalf("by title: %+v %v", byTitle, err)
		}
	})

	t.Run("missing ids return ErrNotFound", func(t *testing.T) {
		repo := newRepo(t)
		if _, err := repo.Get(ctx, 999); !errors.Is(err, course.ErrNotFound) {
			t.Errorf("Get: %v", err)
		}
		if _, err := repo.GetByTitle(ctx, "nope"); !errors.Is(err, course.ErrNotFound) {
			t.Errorf("GetByTitle: %v", err)
		}
		if _, err := repo.Update(ctx, course.Course{ID: 999, Title: "x"}); !errors.Is(err, course.ErrNotFound) {
			t.Errorf("Update: %v", err)
		}
		if err := repo.Delete(ctx, 999); !errors.Is(err, course.ErrNotFound) {
			t.Errorf("Delete: %v", err)
		}
	})

	t.Run("update and delete", func(t *testing.T) {
		repo := newRepo(t)
		c, _ := repo.Get(ctx, 1)
		c.Price = 1
		if _, err := repo.Update(ctx, c); err != nil {
			t.Fatal(err)
		}
		if got, _ := repo.Get(ctx, 1); got.Price != 1 || got.Title != c.Title {
			t.Fatalf("after update: %+v", got)
		}
		if err := repo.Delete(ctx, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Get(ctx, 1); !errors.Is(err, course.ErrNotFound) {
			t.Fatalf("after delete: %v", err)
		}
	})
}
