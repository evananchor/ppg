package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fadhilkurnia/ppg-dashboard/internal/bulk"
)

// PencapaianBulk adapts *PencapaianStore to bulk.Importer, bulk.Exporter
// and bulk.Deleter. The natural key is (student_id, materi_ajar_id) — the
// same pair the pencapaian table treats as unique.
type PencapaianBulk struct {
	db        *sql.DB
	pencapaian *PencapaianStore
}

func NewPencapaianBulk(db *sql.DB, p *PencapaianStore) *PencapaianBulk {
	return &PencapaianBulk{db: db, pencapaian: p}
}

func (b *PencapaianBulk) Name() string { return "pencapaian" }

func (b *PencapaianBulk) Headers() []string {
	return []string{
		"studentId", "studentName", "materiAjarId",
		"tingkat", "nomor", "tema", "subTema", "rincian", "materi", "semester",
		"status", "nilaiAngka", "tanggal", "catatan",
	}
}

// ParseRow validates one CSV row. The human-readable columns (studentName,
// tingkat, nomor, tema, subTema, materi, semester) are informational and
// ignored; ids are the only identity, matching what the exporter writes.
func (b *PencapaianBulk) ParseRow(rec map[string]string) (PencapaianInput, error) {
	studentID := strings.TrimSpace(pickFirst(rec, "studentId", "StudentId"))
	if studentID == "" {
		return PencapaianInput{}, errors.New("studentId is empty")
	}
	materiID := strings.TrimSpace(pickFirst(rec, "materiAjarId", "MateriAjarId"))
	if materiID == "" {
		return PencapaianInput{}, errors.New("materiAjarId is empty")
	}

	statusRaw := strings.ToLower(strings.TrimSpace(pickFirst(rec, "status")))
	status, err := normalisePencapaianStatus(statusRaw)
	if err != nil {
		return PencapaianInput{}, err
	}

	var nilai *int
	if raw := strings.TrimSpace(pickFirst(rec, "nilaiAngka", "NilaiAngka")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > 100 {
			return PencapaianInput{}, fmt.Errorf("nilaiAngka %q: want integer 0-100", raw)
		}
		nilai = &n
	}

	var tanggal *string
	if raw := strings.TrimSpace(pickFirst(rec, "tanggal", "Tanggal")); raw != "" {
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return PencapaianInput{}, fmt.Errorf("tanggal %q: want YYYY-MM-DD", raw)
		}
		tanggal = &[]string{t.Format("2006-01-02")}[0]
	}

	catatan := strings.TrimSpace(pickFirst(rec, "catatan", "Catatan"))
	if len(catatan) > 2000 {
		return PencapaianInput{}, fmt.Errorf("catatan: max 2000 characters")
	}

	return PencapaianInput{
		StudentID:    studentID,
		MateriAjarID: materiID,
		Status:       status,
		NilaiAngka:   nilai,
		Tanggal:      tanggal,
		Catatan:      nilIfEmpty(catatan),
	}, nil
}

// Upsert matches on (student_id, materi_ajar_id). Create mode rejects
// duplicates; upsert mode updates the existing row.
func (b *PencapaianBulk) Upsert(ctx context.Context, in PencapaianInput, mode bulk.Mode) (string, bool, error) {
	existingID, err := b.idByKey(ctx, in.StudentID, in.MateriAjarID)
	if err != nil {
		return "", false, err
	}
	if existingID != "" && mode != bulk.ModeUpsert {
		return "", false, fmt.Errorf("duplicate pencapaian for student %q and materi %q", in.StudentID, in.MateriAjarID)
	}
	created, err := b.pencapaian.Upsert(ctx, in)
	if err != nil {
		return "", false, err
	}
	return created.ID, existingID == "", nil
}

func (b *PencapaianBulk) StreamRows(ctx context.Context, q url.Values, write func([]string) error) error {
	const page = 500
	offset := 0

	where := "1=1"
	var args []any
	if v := strings.TrimSpace(q.Get("studentId")); v != "" {
		where = "p.student_id = ?"
		args = append(args, v)
	}

	for {
		rows, err := b.db.QueryContext(ctx, `
			SELECT p.id, p.student_id, s.name, p.materi_ajar_id,
			       t.nama, m.nomor, m.tema, m.sub_tema, m.rincian, m.materi, m.semester,
			       p.status, p.nilai_angka, p.tanggal, p.catatan
			FROM pencapaian p
			JOIN students s ON s.id = p.student_id
			JOIN materi_ajar m ON m.id = p.materi_ajar_id
			JOIN tingkat t ON t.id = m.tingkat_id
			WHERE `+where+`
			ORDER BY p.student_id ASC, p.materi_ajar_id ASC
			LIMIT ? OFFSET ?`, append(args, page, offset)...)
		if err != nil {
			return err
		}
		n := 0
		for rows.Next() {
			n++
			var id, studentID, studentName, materiID, tingkat string
			var nomor int
			var tema, subTema, materi string
			var rincian *string
			var semester *int
			var status string
			var nilai *int
			var tanggal, catatan *string
			if err := rows.Scan(&id, &studentID, &studentName, &materiID,
				&tingkat, &nomor, &tema, &subTema, &rincian, &materi, &semester,
				&status, &nilai, &tanggal, &catatan); err != nil {
				rows.Close()
				return err
			}
			nilaiRaw := ""
			if nilai != nil {
				nilaiRaw = strconv.Itoa(*nilai)
			}
			semesterRaw := ""
			if semester != nil {
				semesterRaw = strconv.Itoa(*semester)
			}
			if err := write([]string{
				studentID, studentName, materiID,
				tingkat, strconv.Itoa(nomor), tema, subTema, strOrEmpty(rincian), materi, semesterRaw,
				status, nilaiRaw, strOrEmpty(tanggal), strOrEmpty(catatan),
			}); err != nil {
				rows.Close()
				return err
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		if n < page {
			return nil
		}
		offset += page
	}
}

// BulkDelete: archive is unsupported (no status column to flip), hard runs
// DELETE by pencapaian id.
func (b *PencapaianBulk) BulkDelete(ctx context.Context, ids []string, mode bulk.DeleteMode) []bulk.DeleteResult {
	out := make([]bulk.DeleteResult, 0, len(ids))
	if mode == bulk.DeleteModeArchive {
		for _, id := range ids {
			out = append(out, bulk.DeleteResult{
				ID: strings.TrimSpace(id), Outcome: bulk.OutcomeFailed,
				Error: "pencapaian do not support archive; use mode=hard",
			})
		}
		return out
	}
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			out = append(out, bulk.DeleteResult{ID: raw, Outcome: bulk.OutcomeFailed, Error: "empty id"})
			continue
		}
		err := b.pencapaian.Delete(ctx, id)
		if err != nil {
			out = append(out, bulk.DeleteResult{ID: id, Outcome: failOrSkip(err), Error: errMessage(err)})
			continue
		}
		out = append(out, bulk.DeleteResult{ID: id, Outcome: bulk.OutcomeUpdated})
	}
	return out
}

func (b *PencapaianBulk) idByKey(ctx context.Context, studentID, materiID string) (string, error) {
	var id string
	err := b.db.QueryRowContext(ctx,
		`SELECT id FROM pencapaian WHERE student_id = ? AND materi_ajar_id = ? LIMIT 1`,
		studentID, materiID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func normalisePencapaianStatus(raw string) (string, error) {
	for _, s := range []string{"belum", "proses", "tuntas"} {
		if strings.EqualFold(raw, s) {
			return s, nil
		}
	}
	return "", fmt.Errorf("status %q: want belum, proses, or tuntas", raw)
}
