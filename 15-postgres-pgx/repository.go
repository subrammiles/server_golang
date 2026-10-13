package pgdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/subrammiles/server_golang/12-rest-api/course"
)

// Repository implements lesson 12's course.Repository on PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

var _ course.Repository = (*Repository)(nil)

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const columns = `id, title, category, duration_hours, price, created_at`

var orderBy = map[string]string{
	"title": "title ASC", "-title": "title DESC",
	"price": "price ASC", "-price": "price DESC",
	"durationHours": "duration_hours ASC", "-durationHours": "duration_hours DESC",
}

func (r *Repository) List(ctx context.Context, q course.Query) ([]course.Course, int, error) {
	// PostgreSQL parameters are numbered: $1, $2... ILIKE = case-insensitive LIKE.
	// COUNT(*) OVER () returns the total match count on every row: one query instead of two.
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT %s, COUNT(*) OVER () AS total FROM courses
		WHERE ($1 = '' OR category = $1) AND ($2 = '' OR title ILIKE '%%' || $2 || '%%')
		ORDER BY %s LIMIT $3 OFFSET $4`, columns, orderBy[q.Sort]),
		q.Category, q.Q, q.Size, q.Page*q.Size)
	if err != nil {
		return nil, 0, err
	}
	total := 0
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (course.Course, error) {
		var c course.Course
		err := row.Scan(&c.ID, &c.Title, &c.Category, &c.DurationHours, &c.Price, &c.CreatedAt, &total)
		return c, err
	})
	if err != nil {
		return nil, 0, err
	}
	if total == 0 && q.Page > 0 { // page past the end: still report the real total
		err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM courses WHERE ($1 = '' OR category = $1) AND ($2 = '' OR title ILIKE '%' || $2 || '%')`,
			q.Category, q.Q).Scan(&total)
	}
	return items, total, err
}

func (r *Repository) Get(ctx context.Context, id int64) (course.Course, error) {
	return r.one(ctx, `WHERE id = $1`, id)
}

func (r *Repository) GetByTitle(ctx context.Context, title string) (course.Course, error) {
	return r.one(ctx, `WHERE lower(title) = lower($1)`, title)
}

func (r *Repository) Create(ctx context.Context, in course.Input) (course.Course, error) {
	// RETURNING gives back the generated id and timestamp without a second query
	return r.scanOne(r.pool.QueryRow(ctx,
		`INSERT INTO courses (title, category, duration_hours, price) VALUES ($1, $2, $3, $4) RETURNING `+columns,
		in.Title, in.Category, in.DurationHours, in.Price))
}

func (r *Repository) Update(ctx context.Context, c course.Course) (course.Course, error) {
	return r.scanOne(r.pool.QueryRow(ctx,
		`UPDATE courses SET title = $1, category = $2, duration_hours = $3, price = $4 WHERE id = $5 RETURNING `+columns,
		c.Title, c.Category, c.DurationHours, c.Price, c.ID))
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return course.ErrNotFound
	}
	return nil
}

func (r *Repository) one(ctx context.Context, where string, arg any) (course.Course, error) {
	return r.scanOne(r.pool.QueryRow(ctx, `SELECT `+columns+` FROM courses `+where, arg))
}

func (r *Repository) scanOne(row pgx.Row) (course.Course, error) {
	var c course.Course
	err := row.Scan(&c.ID, &c.Title, &c.Category, &c.DurationHours, &c.Price, &c.CreatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return c, course.ErrNotFound
	case isUniqueViolation(err):
		return c, course.ErrDuplicate
	}
	c.CreatedAt = c.CreatedAt.UTC()
	return c, err
}

// PostgreSQL reports errors with SQLSTATE codes: 23505 = unique_violation, 23503 = foreign_key_violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
