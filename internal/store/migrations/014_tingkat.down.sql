DROP INDEX IF EXISTS idx_students_tingkat;
ALTER TABLE students DROP COLUMN tingkat_id;
DROP TABLE IF EXISTS tingkat;
