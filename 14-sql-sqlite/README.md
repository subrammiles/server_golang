# 14: SQL with database/sql (SQLite)

`database/sql` is Go's standard database API. Drivers plug in through a blank import. This lesson uses
`modernc.org/sqlite`, a **pure-Go** SQLite: no C compiler, no database server.

## Concepts
- **`*sql.DB` is a connection pool**, safe for concurrent use. Open it once and tune it with `SetMaxOpenConns`,
  `SetMaxIdleConns`, `SetConnMaxLifetime`. `sql.Open` doesn't connect; `PingContext` does.
- **Always pass `ctx`**: `QueryContext`, `QueryRowContext`, `ExecContext`, so queries are cancelled with the request.
- **`rows.Close()`** (defer it) and **`rows.Err()`**. Forgetting them leaks connections.
- **`sql.ErrNoRows`** becomes the domain `ErrNotFound`. Driver errors (unique violation) become `ErrDuplicate`
  (`mapError`). Upper layers never see SQL details.
- **Parameters (`?`) everywhere**. `FindUnsafe` shows **SQL injection** leaking every row. Column names in `ORDER BY` come from a **whitelist**.
- **Transactions**: `BeginTx`, `defer tx.Rollback()`, `Commit()` (`inTx`). `ImportAll` is all-or-nothing.
- **Prepared statements** for repeated inserts.
- **Migrations** embedded in the binary (`//go:embed migrations/*.sql`), applied once each and tracked in `schema_migrations`.
  Real projects use golang-migrate, goose or atlas.
- **Repository contract**: the same `coursetest.RunContract` as lesson 12, so lesson 12's whole API runs on SQLite unchanged.
- Beyond plain `database/sql`: **sqlc** (generate type-safe Go from SQL), **sqlx** (struct scanning), **GORM** (ORM).

## Run
```bash
go run ./14-sql-sqlite/cmd/server          # creates courses.db; restart and your data is still there
curl localhost:8080/api/courses
go test -v ./14-sql-sqlite
```
