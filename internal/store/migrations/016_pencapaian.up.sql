-- Pencapaian: one row per (student x materi) across ALL tingkat, so a
-- student's achievement history spans every age level. status flows
-- belum -> proses -> tuntas; nilai_angka drives the report grading.

CREATE TABLE pencapaian (
  id             TEXT PRIMARY KEY,
  student_id     TEXT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
  materi_ajar_id TEXT NOT NULL REFERENCES materi_ajar(id) ON DELETE CASCADE,
  status         TEXT NOT NULL DEFAULT 'belum'
                 CHECK (status IN ('belum','proses','tuntas')),
  nilai_angka    INTEGER CHECK (nilai_angka BETWEEN 0 AND 100),
  tanggal        TEXT,
  catatan        TEXT,
  recorded_by    TEXT REFERENCES users(id),
  created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  UNIQUE(student_id, materi_ajar_id)
);

CREATE INDEX idx_pencapaian_student ON pencapaian(student_id);
CREATE INDEX idx_pencapaian_materi  ON pencapaian(materi_ajar_id);
CREATE INDEX idx_pencapaian_tanggal ON pencapaian(materi_ajar_id, tanggal);
