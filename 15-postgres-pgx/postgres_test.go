package pgdb

import (
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	restapi "github.com/subrammiles/server_golang/12-rest-api"
	"github.com/subrammiles/server_golang/12-rest-api/course"
	"github.com/subrammiles/server_golang/12-rest-api/course/coursetest"
)

// One real PostgreSQL in Docker for the whole package (Testcontainers), started on first use.
var (
	startOnce sync.Once
	container *postgres.PostgresContainer
	testPool  *pgxpool.Pool
	startErr  error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if testPool != nil {
		testPool.Close()
	}
	if container != nil {
		testcontainers.TerminateContainer(container)
	}
	os.Exit(code)
}

// freshDB returns a pool on a migrated database reset to the seed data. Skips without Docker.
func freshDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	startOnce.Do(func() {
		container, startErr = postgres.Run(ctx, "postgres:17-alpine",
			postgres.WithDatabase("courses"), postgres.WithUsername("test"), postgres.WithPassword("test"),
			postgres.BasicWaitStrategies())
		if startErr != nil {
			return
		}
		url, _ := container.ConnectionString(ctx, "sslmode=disable")
		if testPool, startErr = Connect(ctx, url); startErr == nil {
			startErr = Migrate(ctx, testPool)
		}
	})
	if startErr != nil {
		t.Fatalf("start postgres: %v", startErr)
	}
	if err := Reset(ctx, testPool); err != nil {
		t.Fatal(err)
	}
	return testPool
}

func TestRepositoryContract(t *testing.T) {
	coursetest.RunContract(t, func(t *testing.T) course.Repository { return NewRepository(freshDB(t)) })
}

func TestMigrateIsIdempotent(t *testing.T) {
	pool := freshDB(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
}

func TestJoins(t *testing.T) {
	e := NewEnrollments(freshDB(t))
	ctx := context.Background()
	if _, err := e.Enroll(ctx, 2, 1); err != nil {
		t.Fatal(err)
	}
	students, err := e.StudentsIn(ctx, 1)
	if err != nil || len(students) != 2 || students[0].Name != "Ravi" {
		t.Fatalf("got %+v %v", students, err)
	}
	counts, _ := e.Counts(ctx)
	if counts[0].Students != 2 || counts[1].Students != 0 {
		t.Fatalf("counts %+v", counts)
	}
}

func TestEnrollmentErrors(t *testing.T) {
	e := NewEnrollments(freshDB(t))
	ctx := context.Background()
	if _, err := e.Enroll(ctx, 1, 1); !errors.Is(err, ErrAlreadyEnrolled) {
		t.Errorf("double enrolment: %v", err)
	}
	if _, err := e.Enroll(ctx, 99, 2); !errors.Is(err, ErrUnknownReference) {
		t.Errorf("unknown student: %v", err)
	}
	if _, err := e.Enroll(ctx, 1, 99); !errors.Is(err, ErrUnknownReference) {
		t.Errorf("unknown course: %v", err)
	}
}

// The important one: many concurrent requests for ONE seat. Without FOR UPDATE this overbooks.
func TestRowLockPreventsOverbooking(t *testing.T) {
	pool := freshDB(t)
	ctx := context.Background()
	pool.Exec(ctx, `UPDATE courses SET capacity = 1 WHERE id = 3`)
	for i := range 8 {
		pool.Exec(ctx, `INSERT INTO students (name, email) VALUES ($1, $2)`, "s", strings.Repeat("x", i+1)+"@example.com")
	}
	e := NewEnrollments(pool)

	var wg sync.WaitGroup
	var mu sync.Mutex
	wins, full := 0, 0
	for student := int64(1); student <= 10; student++ {
		wg.Go(func() {
			_, err := e.Enroll(ctx, student, 3)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				wins++
			case errors.Is(err, ErrCourseFull):
				full++
			default:
				t.Errorf("unexpected: %v", err)
			}
		})
	}
	wg.Wait()
	if wins != 1 || full != 9 {
		t.Fatalf("wins %d, full %d: overbooked!", wins, full)
	}
}

func TestLesson12APIRunsOnPostgres(t *testing.T) {
	api := restapi.NewRouter(NewRepository(freshDB(t)), slog.New(slog.DiscardHandler))
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest("GET", "/api/courses?sort=-price&size=1", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"totalElements":3`) || !strings.Contains(rec.Body.String(), "Data Structures") {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}
