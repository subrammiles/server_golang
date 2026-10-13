// Package sqlitedb shows database/sql, Go's standard database API, with SQLite.
// database/sql is driver-agnostic: the same code style works for PostgreSQL, MySQL, SQL Server...
package sqlitedb

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	_ "modernc.org/sqlite" // the blank import REGISTERS the "sqlite" driver; pure Go, no cgo needed
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Open returns a *sql.DB: NOT a single connection but a concurrency-safe POOL. Create it once at startup.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	// SQLite allows one writer at a time; a single connection avoids "database is locked" errors.
	// For a server database you would set e.g. SetMaxOpenConns(25), SetMaxIdleConns(25), SetConnMaxLifetime(5*time.Minute).
	db.SetMaxOpenConns(1)
	if err := db.PingContext(context.Background()); err != nil { // sql.Open doesn't connect; Ping does
		db.Close()
		return nil, err
	}
	return db, nil
}

// Migrate applies embedded migrations in order, each exactly once, recording them in schema_migrations.
// (Tools like golang-migrate, goose or atlas do this with more features.)
func Migrate(ctx context.Context, db *sql.DB) (applied int, err error) {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
		return 0, err
	}
	names, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return 0, err
	}
	sort.Strings(names)
	for _, name := range names {
		version := strings.TrimSuffix(strings.TrimPrefix(name, "migrations/"), ".sql")
		var exists int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&exists); err != nil {
			return applied, err
		}
		if exists > 0 {
			continue
		}
		body, _ := migrationFiles.ReadFile(name)
		if err := inTx(ctx, db, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, string(body)); err != nil {
				return fmt.Errorf("migration %s: %w", version, err)
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, version)
			return err
		}); err != nil {
			return applied, err
		}
		applied++
	}
	return applied, nil
}

// inTx is the standard TRANSACTION pattern: Begin, defer Rollback (a no-op after Commit), Commit.
func inTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
