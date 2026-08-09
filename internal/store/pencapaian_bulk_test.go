package store

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fadhilkurnia/ppg-dashboard/internal/bulk"
)

func newPencapaianBulkDB(t *testing.T) (*PencapaianBulk, string, string) {
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
	student, err := NewStudents(db).Create(ctx, sampleInput("Bulk Kid"))
	if err != nil {
		t.Fatalf("create student: %v", err)
	}
	materis, err := NewKurikulum(db).ListMateriAjar(ctx, MateriAjarListParams{})
	if err != nil || len(materis) == 0 {
		t.Fatalf("list materi: %v (n=%d)", err, len(materis))
	}
	return NewPencapaianBulk(db, NewPencapaian(db)), student.ID, materis[0].ID
}

func pencapaianRec() map[string]string {
	return map[string]string{
		"studentId":    "",
		"studentName":  "Bulk Kid",
		"materiAjarId": "",
		"status":       "proses",
		"nilaiAngka":   "85",
		"tanggal":      "2026-08-01",
		"catatan":      "lancar",
	}
}

func TestPencapaianParseRow(t *testing.T) {
	b, studentID, materiID := newPencapaianBulkDB(t)
	rec := pencapaianRec()
	rec["studentId"] = studentID
	rec["materiAjarId"] = materiID

	in, err := b.ParseRow(rec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if in.Status != "proses" || in.NilaiAngka == nil || *in.NilaiAngka != 85 {
		t.Errorf("unexpected parse: %+v", in)
	}
	if in.Tanggal == nil || *in.Tanggal != "2026-08-01" {
		t.Errorf("tanggal = %v, want 2026-08-01", in.Tanggal)
	}

	cases := []struct {
		name string
		mut  func(map[string]string)
	}{
		{"empty studentId", func(r map[string]string) { r["studentId"] = "" }},
		{"empty materiAjarId", func(r map[string]string) { r["materiAjarId"] = "" }},
		{"bad status", func(r map[string]string) { r["status"] = "done" }},
		{"nilai > 100", func(r map[string]string) { r["nilaiAngka"] = "101" }},
		{"nilai not int", func(r map[string]string) { r["nilaiAngka"] = "abc" }},
		{"bad tanggal", func(r map[string]string) { r["tanggal"] = "01/08/2026" }},
		{"catatan too long", func(r map[string]string) { r["catatan"] = strings.Repeat("x", 2001) }},
	}
	for _, c := range cases {
		r := pencapaianRec()
		r["studentId"] = studentID
		r["materiAjarId"] = materiID
		c.mut(r)
		if _, err := b.ParseRow(r); err == nil {
			t.Errorf("%s: want error, got nil", c.name)
		}
	}
}

func TestPencapaianUpsertCreateDuplicate(t *testing.T) {
	b, studentID, materiID := newPencapaianBulkDB(t)
	ctx := context.Background()
	in := PencapaianInput{
		StudentID: studentID, MateriAjarID: materiID,
		Status: "tuntas",
	}

	id, created, err := b.Upsert(ctx, in, bulk.ModeCreate)
	if err != nil || !created || id == "" {
		t.Fatalf("create: id=%q created=%v err=%v", id, created, err)
	}
	if _, _, err := b.Upsert(ctx, in, bulk.ModeCreate); err == nil {
		t.Fatal("duplicate create: want error, got nil")
	}

	in.Status = "proses"
	id2, created, err := b.Upsert(ctx, in, bulk.ModeUpsert)
	if err != nil || created || id2 != id {
		t.Fatalf("upsert: id=%q want %q created=%v err=%v", id2, id, created, err)
	}

	rows, err := b.pencapaian.ListByStudent(ctx, in.StudentID)
	if err != nil || len(rows) != 1 || rows[0].Status != "proses" {
		t.Fatalf("after upsert: err=%v n=%d status=%q", err, len(rows), rows[0].Status)
	}
}

func TestPencapaianStreamRowsFilter(t *testing.T) {
	b, studentID, materiID := newPencapaianBulkDB(t)
	ctx := context.Background()
	in := PencapaianInput{
		StudentID: studentID, MateriAjarID: materiID,
		Status: "belum",
	}
	if _, _, err := b.Upsert(ctx, in, bulk.ModeCreate); err != nil {
		t.Fatalf("create: %v", err)
	}

	var lines [][]string
	err := b.StreamRows(ctx, url.Values{"studentId": []string{in.StudentID}}, func(l []string) error {
		lines = append(lines, l)
		return nil
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("streamed %d rows, want 1", len(lines))
	}
	if lines[0][0] != in.StudentID || lines[0][2] != in.MateriAjarID {
		t.Errorf("unexpected ids: %v", lines[0])
	}
	if lines[0][1] != "Bulk Kid" {
		t.Errorf("studentName = %q, want Bulk Kid", lines[0][1])
	}
	if lines[0][9] != "belum" {
		t.Errorf("status = %q, want belum", lines[0][9])
	}
	if lines[0][5] == "" || lines[0][7] == "" || lines[0][3] == "" {
		t.Errorf("materi detail columns empty: %v", lines[0])
	}

	var none [][]string
	err = b.StreamRows(ctx, url.Values{"studentId": []string{"nope"}}, func(l []string) error {
		none = append(none, l)
		return nil
	})
	if err != nil || len(none) != 0 {
		t.Fatalf("filter mismatch: n=%d err=%v", len(none), err)
	}
}
