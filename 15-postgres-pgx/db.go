// Package pgdb uses PostgreSQL through pgx, the most popular Go driver: fast, PostgreSQL-specific features,
// and its own connection pool (pgxpool). (pgx also works as a database/sql driver if you prefer that API.)
package pgdb

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/subrammiles/server_golang/12-rest-api/course"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Connect creates a POOL from a URL like postgres://user:pass@host:5432/db (from an env var, never hard-coded).
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// Migrate applies embedded migrations once each. A PostgreSQL advisory lock makes it safe even when
// several instances of the app start at the same time.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(42)`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
			return err
		}
		names, _ := fs.Glob(migrationFiles, "migrations/*.sql")
		sort.Strings(names)
		for _, name := range names {
			version := strings.TrimSuffix(strings.TrimPrefix(name, "migrations/"), ".sql")
			tag, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, version)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				continue // already applied
			}
			body, _ := migrationFiles.ReadFile(name)
			if _, err := tx.Exec(ctx, string(body)); err != nil {
				return fmt.Errorf("migration %s: %w", version, err)
			}
		}
		return nil
	})
}

// Reset empties the tables and loads sample data (used by the demo server and the tests).
func Reset(ctx context.Context, pool *pgxpool.Pool) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `TRUNCATE enrollments, students, courses RESTART IDENTITY CASCADE`); err != nil {
			return err
		}
		// A BATCH sends many statements in one network round trip
		batch := &pgx.Batch{}
		for _, c := range course.Seed() {
			batch.Queue(`INSERT INTO courses (title, category, duration_hours, price) VALUES ($1, $2, $3, $4)`,
				c.Title, c.Category, c.DurationHours, c.Price)
		}
		batch.Queue(`INSERT INTO students (name, email) VALUES ('Subbu', 'subbu@example.com'), ('Ravi', 'ravi@example.com')`)
		batch.Queue(`INSERT INTO enrollments (student_id, course_id) VALUES (1, 1)`)
		return tx.SendBatch(ctx, batch).Close()
	})
}
