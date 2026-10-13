package sqlitedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/subrammiles/server_golang/12-rest-api/course"
)

// Repository implements lesson 12's course.Repository with SQL.
type Repository struct {
	db *sql.DB
}

var _ course.Repository = (*Repository)(nil) // compile-time interface check

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

const columns = `id, title, category, duration_hours, price, created_at`

// Identifiers (column names, ASC/DESC) can't be query PARAMETERS, so they come from a WHITELIST.
var orderBy = map[string]string{
	"title": "title ASC", "-title": "title DESC",
	"price": "price ASC", "-price": "price DESC",
	"durationHours": "duration_hours ASC", "-durationHours": "duration_hours DESC",
}

func (r *Repository) List(ctx context.Context, q course.Query) ([]course.Course, int, error) {
	var where []string
	var args []any
	if q.Category != "" {
		where = append(where, "category = ?")
		args = append(args, q.Category)
	}
	if q.Q != "" {
		where = append(where, "title LIKE ?")
		args = append(args, "%"+q.Q+"%")
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM courses "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf("SELECT %s FROM courses %s ORDER BY %s LIMIT ? OFFSET ?", columns, whereSQL, orderBy[q.Sort]),
		append(args, q.Size, q.Page*q.Size)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close() // ALWAYS close rows, or the connection never returns to the pool

	var out []course.Course
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err() // errors during iteration surface here
}

func (r *Repository) Get(ctx context.Context, id int64) (course.Course, error) {
	return r.one(ctx, "WHERE id = ?", id)
}

func (r *Repository) GetByTitle(ctx context.Context, title string) (course.Course, error) {
	return r.one(ctx, "WHERE lower(title) = lower(?)", title)
}

func (r *Repository) Create(ctx context.Context, in course.Input) (course.Course, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO courses (title, category, duration_hours, price, created_at) VALUES (?, ?, ?, ?, ?)`,
		in.Title, in.Category, in.DurationHours, in.Price, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return course.Course{}, mapError(err)
	}
	id, _ := res.LastInsertId()
	return r.Get(ctx, id)
}

func (r *Repository) Update(ctx context.Context, c course.Course) (course.Course, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE courses SET title = ?, category = ?, duration_hours = ?, price = ? WHERE id = ?`,
		c.Title, c.Category, c.DurationHours, c.Price, c.ID)
	if err != nil {
		return course.Course{}, mapError(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return course.Course{}, course.ErrNotFound
	}
	return r.Get(ctx, c.ID)
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM courses WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return course.ErrNotFound
	}
	return nil
}

// ImportAll inserts many courses in ONE TRANSACTION: all of them, or none.
func (r *Repository) ImportAll(ctx context.Context, inputs []course.Input) error {
	return inTx(ctx, r.db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, // PREPARED once, executed many times
			`INSERT INTO courses (title, category, duration_hours, price, created_at) VALUES (?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		now := time.Now().UTC().Format(time.RFC3339Nano)
		for _, in := range inputs {
			if _, err := stmt.ExecContext(ctx, in.Title, in.Category, in.DurationHours, in.Price, now); err != nil {
				return mapError(err)
			}
		}
		return nil
	})
}

func (r *Repository) one(ctx context.Context, where string, arg any) (course.Course, error) {
	c, err := scanCourse(r.db.QueryRowContext(ctx, "SELECT "+columns+" FROM courses "+where, arg))
	if errors.Is(err, sql.ErrNoRows) { // "no row" is an expected outcome: translate to the domain error
		return course.Course{}, course.ErrNotFound
	}
	return c, err
}

type scanner interface{ Scan(dest ...any) error }

func scanCourse(s scanner) (course.Course, error) {
	var c course.Course
	var created string
	if err := s.Scan(&c.ID, &c.Title, &c.Category, &c.DurationHours, &c.Price, &created); err != nil {
		return c, err
	}
	var err error
	c.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return c, err
}

// mapError turns driver-specific errors into domain errors, so upper layers never see SQLite details.
func mapError(err error) error {
	var se *sqlite.Error
	if errors.As(err, &se) && se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		return course.ErrDuplicate
	}
	return err
}
