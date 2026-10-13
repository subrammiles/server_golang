CREATE TABLE courses (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    title          TEXT    NOT NULL,
    category       TEXT    NOT NULL,
    duration_hours INTEGER NOT NULL CHECK (duration_hours > 0),
    price          INTEGER NOT NULL CHECK (price >= 0),
    created_at     TEXT    NOT NULL
);
CREATE UNIQUE INDEX courses_title_unique ON courses (lower(title));
