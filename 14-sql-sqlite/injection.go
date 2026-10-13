package sqlitedb

import (
	"context"
	"database/sql"
)

// FindUnsafe builds SQL by STRING CONCATENATION. NEVER do this: input becomes part of the SQL code.
func FindUnsafe(ctx context.Context, db *sql.DB, title string) ([]string, error) {
	return titles(db.QueryContext(ctx, "SELECT title FROM courses WHERE title = '"+title+"'"))
}

// FindSafe passes input as a PARAMETER: it can never change the meaning of the query.
func FindSafe(ctx context.Context, db *sql.DB, title string) ([]string, error) {
	return titles(db.QueryContext(ctx, "SELECT title FROM courses WHERE title = ?", title))
}

func titles(rows *sql.Rows, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
