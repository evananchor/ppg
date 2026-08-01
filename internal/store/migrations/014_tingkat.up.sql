-- Tingkat (age class) catalogue + per-student manual override column.
-- Auto-assignment computes umur from students.date_of_birth; tingkat_id
-- overrides that when set.

CREATE TABLE tingkat (
  id     TEXT PRIMARY KEY,
  nama   TEXT NOT NULL UNIQUE,
  urutan INTEGER NOT NULL DEFAULT 0,
  umur   INTEGER
);

CREATE INDEX idx_tingkat_urutan ON tingkat(urutan);

-- SQLite allows ADD COLUMN with a FK as long as the default is NULL.
ALTER TABLE students ADD COLUMN tingkat_id TEXT REFERENCES tingkat(id);

CREATE INDEX idx_students_tingkat ON students(tingkat_id);
