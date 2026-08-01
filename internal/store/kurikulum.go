package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/oklog/ulid/v2"
)

type Kurikulum struct {
	db *sql.DB
}

func NewKurikulum(db *sql.DB) *Kurikulum {
	return &Kurikulum{db: db}
}

type Tingkat struct {
	ID     string `json:"id"`
	Nama   string `json:"nama"`
	Urutan int    `json:"urutan"`
	Umur   *int   `json:"umur,omitempty"`
}

type TingkatInput struct {
	Nama   string
	Urutan int
	Umur   *int
}

type MateriAjar struct {
	ID             string  `json:"id"`
	TingkatID      string  `json:"tingkatId"`
	Nomor          int     `json:"nomor"`
	SumberNo       *int    `json:"sumberNo,omitempty"`
	Tema           string  `json:"tema"`
	SubTema        string  `json:"subTema"`
	Rincian        *string `json:"rincian,omitempty"`
	Materi         string  `json:"materi"`
	Cakupan        *string `json:"cakupan,omitempty"`
	Jenis          string  `json:"jenis"`
	TargetSemester *string `json:"targetSemester,omitempty"`
	Deskripsi      *string `json:"deskripsi,omitempty"`
	StatusPromes   *string `json:"statusPromes,omitempty"`
	Keterangan     *string `json:"keterangan,omitempty"`
	Semester       *int    `json:"semester,omitempty"`
}

type MateriAjarInput struct {
	TingkatID      string
	Nomor          int
	SumberNo       *int
	Tema           string
	SubTema        string
	Rincian        *string
	Materi         string
	Cakupan        *string
	Jenis          string
	TargetSemester *string
	Deskripsi      *string
	StatusPromes   *string
	Keterangan     *string
	Semester       *int
}

type MateriAjarListParams struct {
	TingkatID string
	Semester  int
	Tema      string
	Query     string
}

const materiAjarCols = `id, tingkat_id, nomor, sumber_no, tema, sub_tema, rincian, materi,
	cakupan, jenis, target_semester, deskripsi, status_promes, keterangan, semester`

func (k *Kurikulum) ListTingkat(ctx context.Context) ([]Tingkat, error) {
	rows, err := k.db.QueryContext(ctx,
		`SELECT id, nama, urutan, umur FROM tingkat ORDER BY urutan ASC, nama ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Tingkat{}
	for rows.Next() {
		var t Tingkat
		if err := rows.Scan(&t.ID, &t.Nama, &t.Urutan, &t.Umur); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (k *Kurikulum) GetTingkat(ctx context.Context, id string) (*Tingkat, error) {
	var t Tingkat
	err := k.db.QueryRowContext(ctx,
		`SELECT id, nama, urutan, umur FROM tingkat WHERE id = ?`, id).
		Scan(&t.ID, &t.Nama, &t.Urutan, &t.Umur)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (k *Kurikulum) CreateTingkat(ctx context.Context, in TingkatInput) (*Tingkat, error) {
	id := ulid.Make().String()
	if _, err := k.db.ExecContext(ctx,
		`INSERT INTO tingkat (id, nama, urutan, umur) VALUES (?, ?, ?, ?)`,
		id, in.Nama, in.Urutan, in.Umur,
	); err != nil {
		return nil, err
	}
	return k.GetTingkat(ctx, id)
}

func (k *Kurikulum) UpdateTingkat(ctx context.Context, id string, in TingkatInput) (*Tingkat, error) {
	res, err := k.db.ExecContext(ctx,
		`UPDATE tingkat SET nama = ?, urutan = ?, umur = ? WHERE id = ?`,
		in.Nama, in.Urutan, in.Umur, id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return k.GetTingkat(ctx, id)
}

// DeleteTingkat refuses when materi_ajar rows still reference the tingkat.
func (k *Kurikulum) DeleteTingkat(ctx context.Context, id string) error {
	var refCount int
	if err := k.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM materi_ajar WHERE tingkat_id = ?`, id).Scan(&refCount); err != nil {
		return err
	}
	if refCount > 0 {
		return ErrTingkatInUse
	}
	res, err := k.db.ExecContext(ctx, `DELETE FROM tingkat WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

var ErrTingkatInUse = errors.New("tingkat still referenced by materi_ajar")

func (k *Kurikulum) ListMateriAjar(ctx context.Context, p MateriAjarListParams) ([]MateriAjar, error) {
	clauses := []string{"1=1"}
	var args []any
	if p.TingkatID != "" {
		clauses = append(clauses, "tingkat_id = ?")
		args = append(args, p.TingkatID)
	}
	if p.Semester != 0 {
		clauses = append(clauses, "semester = ?")
		args = append(args, p.Semester)
	}
	if p.Tema != "" {
		clauses = append(clauses, "tema = ?")
		args = append(args, p.Tema)
	}
	if q := strings.TrimSpace(p.Query); q != "" {
		clauses = append(clauses, `(tema LIKE ? OR sub_tema LIKE ? OR rincian LIKE ? OR materi LIKE ?)`)
		like := "%" + q + "%"
		args = append(args, like, like, like, like)
	}
	rows, err := k.db.QueryContext(ctx,
		`SELECT `+materiAjarCols+` FROM materi_ajar WHERE `+strings.Join(clauses, " AND ")+
			` ORDER BY semester ASC, nomor ASC`,
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MateriAjar{}
	for rows.Next() {
		m, err := scanMateriAjar(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (k *Kurikulum) GetMateriAjar(ctx context.Context, id string) (*MateriAjar, error) {
	row := k.db.QueryRowContext(ctx,
		`SELECT `+materiAjarCols+` FROM materi_ajar WHERE id = ?`, id)
	m, err := scanMateriAjar(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (k *Kurikulum) CreateMateriAjar(ctx context.Context, in MateriAjarInput) (*MateriAjar, error) {
	if in.Jenis == "" {
		in.Jenis = "baru"
	}
	id := ulid.Make().String()
	_, err := k.db.ExecContext(ctx,
		`INSERT INTO materi_ajar (`+materiAjarCols+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, in.TingkatID, in.Nomor, in.SumberNo, in.Tema, in.SubTema, in.Rincian, in.Materi,
		in.Cakupan, in.Jenis, in.TargetSemester, in.Deskripsi, in.StatusPromes, in.Keterangan, in.Semester)
	if err != nil {
		return nil, err
	}
	return k.GetMateriAjar(ctx, id)
}

func (k *Kurikulum) UpdateMateriAjar(ctx context.Context, id string, in MateriAjarInput) (*MateriAjar, error) {
	if in.Jenis == "" {
		in.Jenis = "baru"
	}
	res, err := k.db.ExecContext(ctx,
		`UPDATE materi_ajar SET tingkat_id=?, nomor=?, sumber_no=?, tema=?, sub_tema=?, rincian=?,
		   materi=?, cakupan=?, jenis=?, target_semester=?, deskripsi=?, status_promes=?, keterangan=?, semester=?
		 WHERE id = ?`,
		in.TingkatID, in.Nomor, in.SumberNo, in.Tema, in.SubTema, in.Rincian, in.Materi,
		in.Cakupan, in.Jenis, in.TargetSemester, in.Deskripsi, in.StatusPromes, in.Keterangan, in.Semester, id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return k.GetMateriAjar(ctx, id)
}

func (k *Kurikulum) DeleteMateriAjar(ctx context.Context, id string) error {
	res, err := k.db.ExecContext(ctx, `DELETE FROM materi_ajar WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanMateriAjar(s scanner) (*MateriAjar, error) {
	var m MateriAjar
	if err := s.Scan(
		&m.ID, &m.TingkatID, &m.Nomor, &m.SumberNo, &m.Tema, &m.SubTema, &m.Rincian, &m.Materi,
		&m.Cakupan, &m.Jenis, &m.TargetSemester, &m.Deskripsi, &m.StatusPromes, &m.Keterangan, &m.Semester,
	); err != nil {
		return nil, err
	}
	return &m, nil
}
