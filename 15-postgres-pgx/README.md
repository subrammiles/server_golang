# 15: PostgreSQL with pgx (transactions, joins, row locking)

`pgx` is the standard choice for PostgreSQL in Go: fast, with PostgreSQL types, batches, `COPY`, LISTEN/NOTIFY
and its own pool (`pgxpool`).

## Concepts
- **Connection pool** from a `DATABASE_URL` env var, with `MaxConns`. `Ping` at startup.
- **Numbered parameters** `$1, $2`. `ILIKE`. `COUNT(*) OVER ()` gives the total and the page in one query.
- **`RETURNING`**: get generated ids and defaults back from `INSERT`/`UPDATE`.
- **`pgx.CollectRows` + `RowToStructByName`**: scan rows into structs by `db` tags.
- **Batches**: many statements in one round trip (`Reset`).
- **Errors**: `pgx.ErrNoRows` becomes `ErrNotFound`. **SQLSTATE codes** `23505` (unique) and `23503` (foreign key) become domain errors.
- **Relational design**: students *-* courses through `enrollments`, foreign keys, a composite primary key, CHECK constraints.
- **Joins**: INNER JOIN (students in a course), LEFT JOIN + GROUP BY (counts including zero).
- **Transactions** with `pgx.BeginFunc` (commit on nil, rollback on error).
- **Race conditions**: `SELECT ... FOR UPDATE` serialises seat-taking. The test fires 10 concurrent enrolments
  at a 1-seat course and exactly one wins.
- **Migrations** under an **advisory lock**, so several app instances starting together don't collide.
- **Testcontainers**: tests start a real `postgres:17-alpine`, run the same **contract** as lessons 12 and 14,
  and skip without Docker.

## Run
```bash
docker compose -f 15-postgres-pgx/docker-compose.yml up -d --wait
DATABASE_URL=postgres://tutorial:tutorial@localhost:55432/courses go run ./15-postgres-pgx/cmd/server
curl 'localhost:8080/api/courses?sort=-price'
go test -v ./15-postgres-pgx          # needs Docker; skipped otherwise
docker compose -f 15-postgres-pgx/docker-compose.yml down
```
