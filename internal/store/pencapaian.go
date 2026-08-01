package store

import (
	"context"
	"database/sql"

	"github.com/oklog/ulid/v2"
)

type PencapaianStore struct {
	db *sql.DB
}

func NewPencapaian(db *sql.DB) *PencapaianStore {
	return &PencapaianStore{db: db}
}

type Pencapaian struct {
	ID           string  `json:"id"`
	StudentID    string  `json:"studentId"`
	MateriAjarID string  `json:"materiAjarId"`
	Status       string  `json:"status"`
	NilaiAngka   *int    `json:"nilaiAngka,omitempty"`
	Tanggal      *string `json:"tanggal,omitempty"`
	Catatan      *string `json:"catatan,omitempty"`
	RecordedBy   *string `json:"recordedBy,omitempty"`
	UpdatedAt    string  `json:"updatedAt"`
}

type PencapaianInput struct {
	StudentID    string
	MateriAjarID string
	Status       string
	NilaiAngka   *int
	Tanggal      *string
	Catatan      *string
	RecordedBy   string
}

const pencapaianCols = `id, student_id, materi_ajar_id, status, nilai_angka, tanggal, catatan, recorded_by, updated_at`

func scanPencapaian(s scanner) (*Pencapaian, error) {
	var p Pencapaian
	if err := s.Scan(
		&p.ID, &p.StudentID, &p.MateriAjarID, &p.Status,
		&p.NilaiAngka, &p.Tanggal, &p.Catatan, &p.RecordedBy, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &p, nil
}

// UpsertPencapaian creates or updates the (student, materi) record.
func (s *PencapaianStore) Upsert(ctx context.Context, in PencapaianInput) (*Pencapaian, error) {
	id := ulid.Make().String()
	var recordedBy any
	if in.RecordedBy != "" {
		recordedBy = in.RecordedBy
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pencapaian (id, student_id, materi_ajar_id, status, nilai_angka, tanggal, catatan, recorded_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(student_id, materi_ajar_id) DO UPDATE SET
		  status = excluded.status,
		  nilai_angka = excluded.nilai_angka,
		  tanggal = excluded.tanggal,
		  catatan = excluded.catatan,
		  recorded_by = excluded.recorded_by,
		  updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')`,
		id, in.StudentID, in.MateriAjarID, in.Status, in.NilaiAngka, in.Tanggal, in.Catatan, recordedBy)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+pencapaianCols+` FROM pencapaian WHERE student_id = ? AND materi_ajar_id = ?`,
		in.StudentID, in.MateriAjarID)
	return scanPencapaian(row)
}

func (s *PencapaianStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM pencapaian WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// PencapaianRow pairs a pencapaian record with its materi (any tingkat) and
// the tingkat metadata — the per-student all-ages view.
type PencapaianRow struct {
	Pencapaian
	Materi        MateriAjar `json:"materi"`
	TingkatNama   string     `json:"tingkatNama"`
	TingkatUrutan int        `json:"tingkatUrutan"`
}

func (s *PencapaianStore) ListByStudent(ctx context.Context, studentID string) ([]PencapaianRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.student_id, p.materi_ajar_id, p.status, p.nilai_angka, p.tanggal, p.catatan, p.recorded_by, p.updated_at,
		       m.id, m.tingkat_id, m.nomor, m.sumber_no, m.tema, m.sub_tema, m.rincian, m.materi,
		       m.cakupan, m.jenis, m.target_semester, m.deskripsi, m.status_promes, m.keterangan, m.semester,
		       t.nama, t.urutan
		FROM pencapaian p
		JOIN materi_ajar m ON m.id = p.materi_ajar_id
		JOIN tingkat t ON t.id = m.tingkat_id
		WHERE p.student_id = ?
		ORDER BY t.urutan ASC, m.semester ASC, m.nomor ASC`,
		studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PencapaianRow{}
	for rows.Next() {
		var r PencapaianRow
		if err := rows.Scan(
			&r.Pencapaian.ID, &r.Pencapaian.StudentID, &r.Pencapaian.MateriAjarID, &r.Pencapaian.Status,
			&r.Pencapaian.NilaiAngka, &r.Pencapaian.Tanggal, &r.Pencapaian.Catatan, &r.Pencapaian.RecordedBy, &r.Pencapaian.UpdatedAt,
			&r.Materi.ID, &r.Materi.TingkatID, &r.Materi.Nomor, &r.Materi.SumberNo,
			&r.Materi.Tema, &r.Materi.SubTema, &r.Materi.Rincian, &r.Materi.Materi,
			&r.Materi.Cakupan, &r.Materi.Jenis, &r.Materi.TargetSemester, &r.Materi.Deskripsi,
			&r.Materi.StatusPromes, &r.Materi.Keterangan, &r.Materi.Semester,
			&r.TingkatNama, &r.TingkatUrutan,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ReportCell aggregates one month's pencapaian nilai for one materi.
type ReportCell struct {
	Month int  `json:"month"`
	Count int  `json:"count"`
	Avg   *int `json:"avg,omitempty"`
}

// ReportRow is one materi_ajar line of the class report (Khayri style).
type ReportRow struct {
	Materi MateriAjar   `json:"materi"`
	Cells  []ReportCell `json:"cells"`
}

// ClassReport returns every materi_ajar of (tingkat, semester) with the
// pencapaian records of that materi bucketed by calendar month.
func (s *PencapaianStore) ClassReport(ctx context.Context, tingkatID string, semester int) ([]ReportRow, error) {
	mCols := "m.id, m.tingkat_id, m.nomor, m.sumber_no, m.tema, m.sub_tema, m.rincian, m.materi, " +
		"m.cakupan, m.jenis, m.target_semester, m.deskripsi, m.status_promes, m.keterangan, m.semester"
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+mCols+`,
		       COALESCE(p.month, 0) AS month,
		       COUNT(p.id) AS cnt,
		       CAST(AVG(p.nilai_angka) AS INTEGER) AS avg_nilai
		FROM materi_ajar m
		LEFT JOIN (
		  SELECT materi_ajar_id, CAST(strftime('%m', tanggal) AS INTEGER) AS month,
		         id, nilai_angka
		  FROM pencapaian
		) p ON p.materi_ajar_id = m.id
		WHERE m.tingkat_id = ? AND m.semester = ?
		GROUP BY m.id, p.month
		ORDER BY m.nomor ASC, p.month ASC`,
		tingkatID, semester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ReportRow{}
	for rows.Next() {
		var m MateriAjar
		var month, cnt int
		var avg *int
		if err := rows.Scan(
			&m.ID, &m.TingkatID, &m.Nomor, &m.SumberNo, &m.Tema, &m.SubTema, &m.Rincian, &m.Materi,
			&m.Cakupan, &m.Jenis, &m.TargetSemester, &m.Deskripsi, &m.StatusPromes, &m.Keterangan, &m.Semester,
			&month, &cnt, &avg,
		); err != nil {
			return nil, err
		}
		if len(out) == 0 || out[len(out)-1].Materi.ID != m.ID {
			out = append(out, ReportRow{Materi: m, Cells: []ReportCell{}})
		}
		if month != 0 {
			last := &out[len(out)-1]
			last.Cells = append(last.Cells, ReportCell{Month: month, Count: cnt, Avg: avg})
		}
	}
	return out, rows.Err()
}
