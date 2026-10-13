package pgdb

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCourseFull       = errors.New("course is full")
	ErrAlreadyEnrolled  = errors.New("student already enrolled")
	ErrUnknownReference = errors.New("unknown student or course")
)

type Enrollments struct {
	pool *pgxpool.Pool
}

func NewEnrollments(pool *pgxpool.Pool) *Enrollments { return &Enrollments{pool: pool} }

// Enroll takes a seat in a course.
//
// RACE CONDITION: two requests for the LAST seat run at the same time. Both count "1 seat left" and
// both insert, so the course is overbooked. SELECT ... FOR UPDATE LOCKS the course row until the
// transaction ends: the second request WAITS, then sees the seat is gone.
func (e *Enrollments) Enroll(ctx context.Context, studentID, courseID int64) (seatsLeft int, err error) {
	err = pgx.BeginFunc(ctx, e.pool, func(tx pgx.Tx) error { // commit on nil, rollback on error
		var capacity int
		if err := tx.QueryRow(ctx, `SELECT capacity FROM courses WHERE id = $1 FOR UPDATE`, courseID).Scan(&capacity); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUnknownReference
			}
			return err
		}
		var taken int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM enrollments WHERE course_id = $1`, courseID).Scan(&taken); err != nil {
			return err
		}
		if taken >= capacity {
			return ErrCourseFull
		}
		if _, err := tx.Exec(ctx, `INSERT INTO enrollments (student_id, course_id) VALUES ($1, $2)`, studentID, courseID); err != nil {
			switch {
			case isUniqueViolation(err):
				return ErrAlreadyEnrolled
			case isForeignKeyViolation(err):
				return ErrUnknownReference
			}
			return err
		}
		seatsLeft = capacity - taken - 1
		return nil
	})
	return seatsLeft, err
}

type Student struct {
	ID    int64  `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

// StudentsIn: an INNER JOIN across the many-to-many table, scanned straight into structs by `db` tags.
func (e *Enrollments) StudentsIn(ctx context.Context, courseID int64) ([]Student, error) {
	rows, err := e.pool.Query(ctx, `
		SELECT s.id, s.name, s.email
		FROM students s JOIN enrollments e ON e.student_id = s.id
		WHERE e.course_id = $1 ORDER BY s.name`, courseID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Student])
}

type CourseCount struct {
	Title    string `db:"title"`
	Students int    `db:"students"`
}

// Counts: LEFT JOIN + GROUP BY includes courses with zero students.
func (e *Enrollments) Counts(ctx context.Context) ([]CourseCount, error) {
	rows, err := e.pool.Query(ctx, `
		SELECT c.title, COUNT(e.student_id)::int AS students
		FROM courses c LEFT JOIN enrollments e ON e.course_id = c.id
		GROUP BY c.id, c.title ORDER BY c.id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[CourseCount])
}
