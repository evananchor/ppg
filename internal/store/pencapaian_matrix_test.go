package store

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/fadhilkurnia/ppg-dashboard/internal/bulk"
)

func newMatrixDB(t *testing.T) (*PencapaianStore, string, string, MateriAjar, int) {
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
	ctx := context.Background()
	if _, err := SeedKurikulum(ctx, db); err != nil {
		t.Fatalf("seed kurikulum: %v", err)
	}
	student, err := NewStudents(db).Create(ctx, sampleInput("Matrix Kid"))
	if err != nil {
		t.Fatalf("create student: %v", err)
	}
	materis, err := NewKurikulum(db).ListMateriAjar(ctx, MateriAjarListParams{})
	if err != nil || len(materis) == 0 {
		t.Fatalf("list materi: %v (n=%d)", err, len(materis))
	}
	sem := 1
	if materis[0].Semester != nil {
		sem = *materis[0].Semester
	}
	return NewPencapaian(db), student.ID, materis[0].TingkatID, materis[0], sem
}

func matrixRec(nomor, sem int, status string) map[string]string {
	return map[string]string{
		"No":       strconv.Itoa(nomor),
		"Semester": strconv.Itoa(sem),
		"Tema":     "ALIM",
		"Sub Tema": "A. Baca Tulis",
		"Materi":   "x",
		"Status":   status,
	}
}

func TestMatrixParseRow(t *testing.T) {
	p, studentID, tingkatID, m, sem := newMatrixDB(t)
	imp := NewPencapaianMatrix(p, studentID, tingkatID, sem)

	rec := matrixRec(m.Nomor, sem, "proses")
	rec["Jul"] = "85"
	rec["Okt"] = "90"
	row, err := imp.ParseRow(rec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if row.Nomor != m.Nomor || row.Status != "proses" {
		t.Errorf("unexpected row: %+v", row)
	}
	if row.Cells[7] != 85 || row.Cells[10] != 90 {
		t.Errorf("cells = %v, want {7:85 10:90}", row.Cells)
	}

	// localized en + numeric month headers
	rec2 := matrixRec(m.Nomor, sem, "")
	rec2["Jul"] = ""
	rec2["Aug"] = "72"
	rec2["09"] = "60"
	row2, err := imp.ParseRow(rec2)
	if err != nil {
		t.Fatalf("parse2: %v", err)
	}
	if len(row2.Cells) != 2 || row2.Cells[8] != 72 || row2.Cells[9] != 60 {
		t.Errorf("cells2 = %v, want {8:72 9:60}", row2.Cells)
	}

	cases := []struct {
		name string
		mut  func(map[string]string)
	}{
		{"bad No", func(r map[string]string) { r["No"] = "x" }},
		{"semester mismatch", func(r map[string]string) { r["Semester"] = "2" }},
		{"bad status", func(r map[string]string) { r["Status"] = "done" }},
		{"nilai > 100", func(r map[string]string) { r["Jul"] = "101" }},
		{"nilai not int", func(r map[string]string) { r["Jul"] = "abc" }},
	}
	for _, c := range cases {
		r := matrixRec(m.Nomor, sem, "belum")
		r["Jul"] = "70"
		c.mut(r)
		if _, err := imp.ParseRow(r); err == nil {
			t.Errorf("%s: want error, got nil", c.name)
		}
	}
}

func TestMatrixUpsertDelta(t *testing.T) {
	p, studentID, tingkatID, m, sem := newMatrixDB(t)
	ctx := context.Background()
	imp := NewPencapaianMatrix(p, studentID, tingkatID, sem)

	// 1. blank row -> no write
	blank := MatrixRow{Nomor: m.Nomor, Cells: map[int]int{}}
	id, created, err := imp.Upsert(ctx, blank, bulk.ModeUpsert)
	if err != nil || created || id != "" {
		t.Fatalf("blank: id=%q created=%v err=%v", id, created, err)
	}
	if rows, _ := p.ListByStudent(ctx, studentID); len(rows) != 0 {
		t.Fatalf("blank row wrote %d pencapaian, want 0", len(rows))
	}

	// 2. status only -> create with status, no tanggal
	id, created, err = imp.Upsert(ctx, MatrixRow{Nomor: m.Nomor, Status: "tuntas"}, bulk.ModeUpsert)
	if err != nil || !created || id == "" {
		t.Fatalf("status-only create: id=%q created=%v err=%v", id, created, err)
	}

	// 3. same status, no cells -> no change
	id2, created, err := imp.Upsert(ctx, MatrixRow{Nomor: m.Nomor, Status: "tuntas"}, bulk.ModeUpsert)
	if err != nil || created || id2 != id {
		t.Fatalf("no-change: id=%q want %q created=%v err=%v", id2, id, created, err)
	}

	// 4. give it a dated nilai (June 2024, day 30), then move month -> year kept, day 15
	if _, err := p.Upsert(ctx, PencapaianInput{
		StudentID: studentID, MateriAjarID: m.ID, Status: "tuntas",
		NilaiAngka: ptrInt(80), Tanggal: ptrStr("2024-06-30"),
	}); err != nil {
		t.Fatalf("seed nilai: %v", err)
	}
	id3, created, err := imp.Upsert(ctx, MatrixRow{Nomor: m.Nomor, Status: "tuntas", Cells: map[int]int{10: 90}}, bulk.ModeUpsert)
	if err != nil || created || id3 != id {
		t.Fatalf("month move: id=%q created=%v err=%v", id3, created, err)
	}
	rows, _ := p.ListByStudent(ctx, studentID)
	if len(rows) != 1 || rows[0].NilaiAngka == nil || *rows[0].NilaiAngka != 90 {
		t.Fatalf("nilai not updated: %+v", rows)
	}
	if rows[0].Tanggal == nil || *rows[0].Tanggal != "2024-10-15" {
		t.Fatalf("tanggal = %v, want 2024-10-15 (year preserved, day 15)", rows[0].Tanggal)
	}

	// 5. same month + same value -> untouched
	_, _, err = imp.Upsert(ctx, MatrixRow{Nomor: m.Nomor, Status: "tuntas", Cells: map[int]int{10: 90}}, bulk.ModeUpsert)
	if err != nil {
		t.Fatalf("re-import same: %v", err)
	}
	rows, _ = p.ListByStudent(ctx, studentID)
	if rows[0].Tanggal == nil || *rows[0].Tanggal != "2024-10-15" {
		t.Fatalf("same-cell import rewrote tanggal: %v", rows[0].Tanggal)
	}

	// 6. status change only -> tanggal preserved
	_, _, err = imp.Upsert(ctx, MatrixRow{Nomor: m.Nomor, Status: "proses", Cells: map[int]int{10: 90}}, bulk.ModeUpsert)
	if err != nil {
		t.Fatalf("status change: %v", err)
	}
	rows, _ = p.ListByStudent(ctx, studentID)
	if rows[0].Status != "proses" || *rows[0].Tanggal != "2024-10-15" {
		t.Fatalf("status change: status=%q tanggal=%v", rows[0].Status, rows[0].Tanggal)
	}

	// 7. unknown nomor -> error
	_, _, err = imp.Upsert(ctx, MatrixRow{Nomor: 99999, Status: "tuntas"}, bulk.ModeUpsert)
	if err == nil {
		t.Fatal("unknown nomor: want error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unknown nomor error = %q, want 'not found'", err.Error())
	}
}

func ptrInt(n int) *int    { return &n }
func ptrStr(s string) *string { return &s }
