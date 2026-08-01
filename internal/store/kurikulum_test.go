package store

import (
	"context"
	"path/filepath"
	"testing"
)

func seedTestDB(t *testing.T) (*Kurikulum, *PencapaianStore) {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	n, err := SeedKurikulum(context.Background(), db)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n != 0 {
		t.Fatalf("seed count = %d, want 0 (fresh db)", n)
	}
	return NewKurikulum(db), NewPencapaian(db)
}

func TestSeedKurikulum(t *testing.T) {
	k, _ := seedTestDB(t)
	ts, err := k.ListTingkat(context.Background())
	if err != nil {
		t.Fatalf("list tingkat: %v", err)
	}
	if len(ts) != 13 {
		t.Fatalf("tingkat = %d, want 13", len(ts))
	}
	ms, err := k.ListMateriAjar(context.Background(), MateriAjarListParams{TingkatID: ts[0].ID, Semester: 1})
	if err != nil {
		t.Fatalf("list materi: %v", err)
	}
	if len(ms) != 45 {
		t.Fatalf("≤6 SMT-1 materi = %d, want 45", len(ms))
	}
	if ms[0].Materi != "Melafalkan huruf hijaiyah" {
		t.Fatalf("first materi = %q, want Melafalkan huruf hijaiyah", ms[0].Materi)
	}
}

func TestPencapaianAndClassReport(t *testing.T) {
	k, p := seedTestDB(t)
	ts, _ := k.ListTingkat(context.Background())
	materi, _ := k.ListMateriAjar(context.Background(), MateriAjarListParams{TingkatID: ts[0].ID, Semester: 1})
	student, _ := NewStudents(k.db).Create(context.Background(), sampleInput("Achieve Kid"))

	tanggal := "2026-02-10"
	nilai := 95
	if _, err := p.Upsert(context.Background(), PencapaianInput{
		StudentID: student.ID, MateriAjarID: materi[0].ID,
		Status: "tuntas", NilaiAngka: &nilai, Tanggal: &tanggal,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	// upsert again on the same (student, materi) must update, not duplicate.
	nilai = 90
	if _, err := p.Upsert(context.Background(), PencapaianInput{
		StudentID: student.ID, MateriAjarID: materi[0].ID,
		Status: "proses", NilaiAngka: &nilai, Tanggal: &tanggal,
	}); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}

	rows, err := p.ListByStudent(context.Background(), student.ID)
	if err != nil {
		t.Fatalf("list by student: %v", err)
	}
	if len(rows) != 1 || rows[0].Status != "proses" {
		t.Fatalf("rows = %+v, want 1 proses row", rows)
	}
	if rows[0].TingkatNama != "≤6 Tahun" {
		t.Fatalf("tingkat nama = %q, want ≤6 Tahun", rows[0].TingkatNama)
	}

	report, err := p.ClassReport(context.Background(), ts[0].ID, 1)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if len(report) != 45 {
		t.Fatalf("report rows = %d, want 45", len(report))
	}
	if len(report[0].Cells) != 1 || report[0].Cells[0].Month != 2 || report[0].Cells[0].Avg == nil || *report[0].Cells[0].Avg != 90 {
		t.Fatalf("report[0].Cells = %+v, want Feb avg 90", report[0].Cells)
	}
}
