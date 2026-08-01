-- Materi ajar catalogue, one row per curriculum item per tingkat x semester.
-- Shape mirrors the 2026 report (Tema | Sub Tema | No | Materi) plus the
-- source columns from Tabel_Materi_per_Jenjang_Umur_per_Semester(1).xlsx.

CREATE TABLE materi_ajar (
  id              TEXT PRIMARY KEY,
  tingkat_id      TEXT NOT NULL REFERENCES tingkat(id) ON DELETE CASCADE,
  nomor           INTEGER NOT NULL,
  sumber_no       INTEGER,
  tema            TEXT NOT NULL,
  sub_tema        TEXT NOT NULL,
  rincian         TEXT,
  materi          TEXT NOT NULL,
  cakupan         TEXT,
  jenis           TEXT NOT NULL DEFAULT 'baru'
                  CHECK (jenis IN ('baru','lanjutan','mengulang')),
  target_semester TEXT,
  deskripsi       TEXT,
  status_promes   TEXT,
  keterangan      TEXT,
  semester        INTEGER CHECK (semester IN (1,2)),
  UNIQUE(tingkat_id, semester, nomor)
);

CREATE INDEX idx_materi_ajar_tingkat  ON materi_ajar(tingkat_id);
CREATE INDEX idx_materi_ajar_tema     ON materi_ajar(tema);
CREATE INDEX idx_materi_ajar_semester ON materi_ajar(semester);
