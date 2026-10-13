package sqlitedb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	restapi "github.com/subrammiles/server_golang/12-rest-api"
	"github.com/subrammiles/server_golang/12-rest-api/course"
	"github.com/subrammiles/server_golang/12-rest-api/course/coursetest"
)

// newDB gives every test its own migrated, seeded database file in a temp dir.
func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := NewRepository(db).ImportAll(context.Background(), course.Seed()); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRepositoryContract(t *testing.T) {
	coursetest.RunContract(t, func(t *testing.T) course.Repository { return NewRepository(newDB(t)) })
}

func TestMigrationsRunOnce(t *testing.T) {
	db := newDB(t)
	n, err := Migrate(context.Background(), db)
	if err != nil || n != 0 {
		t.Fatalf("second run applied %d (err %v), want 0", n, err)
	}
}

func TestConstraintsAndDuplicates(t *testing.T) {
	db := newDB(t)
	repo := NewRepository(db)
	_, err := repo.Create(context.Background(), course.Input{Title: "go fundamentals", Category: "x", DurationHours: 1, Price: 1})
	if !errors.Is(err, course.ErrDuplicate) {
		t.Fatalf("want ErrDuplicate, got %v", err)
	}
	_, err = db.Exec(`INSERT INTO courses (title, category, duration_hours, price, created_at) VALUES ('x', 'c', 0, 1, 'now')`)
	if err == nil || !strings.Contains(err.Error(), "CHECK") {
		t.Fatalf("CHECK constraint not enforced: %v", err)
	}
}

func TestTransactionIsAllOrNothing(t *testing.T) {
	db := newDB(t)
	repo := NewRepository(db)
	err := repo.ImportAll(context.Background(), []course.Input{
		{Title: "One", Category: "c", DurationHours: 1, Price: 1},
		{Title: "Go Fundamentals", Category: "c", DurationHours: 1, Price: 1}, // duplicate -> whole import fails
	})
	if !errors.Is(err, course.ErrDuplicate) {
		t.Fatalf("got %v", err)
	}
	if _, err := repo.GetByTitle(context.Background(), "One"); !errors.Is(err, course.ErrNotFound) {
		t.Fatal("first insert was not rolled back")
	}
}

func TestSQLInjection(t *testing.T) {
	db := newDB(t)
	attack := "x' OR '1'='1"
	leaked, _ := FindUnsafe(context.Background(), db, attack)
	safe, _ := FindSafe(context.Background(), db, attack)
	if len(leaked) != 3 || len(safe) != 0 {
		t.Fatalf("unsafe returned %d rows, safe %d", len(leaked), len(safe))
	}
}

func TestLesson12APIRunsOnSQLite(t *testing.T) {
	api := restapi.NewRouter(NewRepository(newDB(t)), slog.New(slog.DiscardHandler))
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest("POST", "/api/courses",
		strings.NewReader(`{"title":"SQL","category":"data","durationHours":4,"price":10}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest("POST", "/api/courses",
		strings.NewReader(`{"title":"sql","category":"data","durationHours":4,"price":10}`)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: %d", rec.Code)
	}
}
