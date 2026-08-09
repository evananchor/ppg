package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fadhilkurnia/ppg-dashboard/internal/bulk"
)

// MatrixRow is one materi row of the assessment matrix CSV:
//
//	No, Semester, Tema, Sub Tema, Materi, Status, <month columns>
//
// Status is optional (blank = keep existing). Cells maps each filled month
// column to its nilai; blank cells mean "no change".
type MatrixRow struct {
	Nomor  int
	Status string
	Cells  map[int]int // month -> nilai
}

// PencapaianMatrix adapts the assessment matrix CSV to the bulk pipeline.
// Context (student, tingkat, semester) is captured at construction; materi
// rows resolve by nomor within that scope, and writes are delta-only: a row
// that changed nothing is left untouched.
type PencapaianMatrix struct {
	pencapaian *PencapaianStore
	studentID  string
	tingkatID  string
	semester   int
}

func NewPencapaianMatrix(p *PencapaianStore, studentID, tingkatID string, semester int) *PencapaianMatrix {
	return &PencapaianMatrix{pencapaian: p, studentID: studentID, tingkatID: tingkatID, semester: semester}
}

func (m *PencapaianMatrix) Name() string { return "pencapaian-matrix" }

// Headers is the canonical export layout (semester 1 months). Parsing is
// header-name tolerant, so any localized or numeric month columns work.
func (m *PencapaianMatrix) Headers() []string {
	return []string{"No", "Semester", "Tema", "Sub Tema", "Materi", "Status",
		"Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
}

// monthNum maps localized (id/en) and numeric month headers to 1-12.
var monthNum = map[string]int{
	"jan": 1, "januari": 1, "january": 1, "1": 1, "01": 1,
	"feb": 2, "februari": 2, "february": 2, "2": 2, "02": 2,
	"mar": 3, "maret": 3, "march": 3, "3": 3, "03": 3,
	"apr": 4, "april": 4, "4": 4, "04": 4,
	"mei": 5, "may": 5, "5": 5, "05": 5,
	"jun": 6, "juni": 6, "june": 6, "6": 6, "06": 6,
	"jul": 7, "juli": 7, "july": 7, "7": 7, "07": 7,
	"agu": 8, "agustus": 8, "aug": 8, "august": 8, "8": 8, "08": 8,
	"sep": 9, "september": 9, "9": 9, "09": 9,
	"okt": 10, "oktober": 10, "oct": 10, "october": 10, "10": 10,
	"nov": 11, "november": 11, "11": 11,
	"des": 12, "desember": 12, "dec": 12, "december": 12, "12": 12,
}

func parseMonthHeader(h string) (int, bool) {
	n, ok := monthNum[strings.ToLower(strings.TrimSpace(h))]
	return n, ok
}

func (m *PencapaianMatrix) ParseRow(rec map[string]string) (MatrixRow, error) {
	nomor, err := strconv.Atoi(strings.TrimSpace(pickFirst(rec, "No", "Nomor")))
	if err != nil {
		return MatrixRow{}, fmt.Errorf("No: want integer nomor, got %q", pickFirst(rec, "No", "Nomor"))
	}
	if raw := strings.TrimSpace(pickFirst(rec, "Semester")); raw != "" {
		sem, err := strconv.Atoi(raw)
		if err != nil || sem != m.semester {
			return MatrixRow{}, fmt.Errorf("Semester %q does not match selected semester %d", raw, m.semester)
		}
	}
	status := ""
	if raw := strings.TrimSpace(pickFirst(rec, "Status")); raw != "" {
		status, err = normalisePencapaianStatus(strings.ToLower(raw))
		if err != nil {
			return MatrixRow{}, err
		}
	}
	cells := map[int]int{}
	for h, v := range rec {
		mo, ok := parseMonthHeader(h)
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 100 {
			return MatrixRow{}, fmt.Errorf("nilai %q in month %s: want integer 0-100", v, h)
		}
		cells[mo] = n
	}
	return MatrixRow{Nomor: nomor, Status: status, Cells: cells}, nil
}

// Upsert applies the delta: only a row whose status, nilai, or attributed
// month actually changed is written. Unchanged rows return without a write,
// preserving the original tanggal day. Mode is ignored (the matrix is
// inherently upsert-shaped); dry-run is handled by bulk.Process.
func (m *PencapaianMatrix) Upsert(ctx context.Context, row MatrixRow, _ bulk.Mode) (string, bool, error) {
	materiID, err := m.pencapaian.findMateriIDByNomor(ctx, m.tingkatID, row.Nomor, m.semester)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", false, fmt.Errorf("materi nomor %d not found in selected age group/semester", row.Nomor)
		}
		return "", false, err
	}
	existing, err := m.pencapaian.findByKey(ctx, m.studentID, materiID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return "", false, err
	}

	status := "belum"
	if existing != nil {
		status = existing.Status
	}
	if row.Status != "" {
		status = row.Status
	}

	nilai := (*int)(nil)
	tanggal := (*string)(nil)
	if existing != nil {
		nilai = existing.NilaiAngka
		tanggal = existing.Tanggal
	}

	changed := existing == nil && (row.Status != "" || len(row.Cells) > 0)
	if mo, v, ok := pickChangedCell(existing, row.Cells); ok {
		changed = true
		year := time.Now().Year()
		if tanggal != nil {
			if t, err := time.Parse("2006-01-02", *tanggal); err == nil {
				year = t.Year()
			}
		}
		nilai = &v
		tanggal = &[]string{fmt.Sprintf("%04d-%02d-15", year, mo)}[0]
	} else if row.Status != "" && (existing == nil || row.Status != existing.Status) {
		changed = true
	}

	if !changed {
		id := ""
		if existing != nil {
			id = existing.ID
		}
		return id, false, nil
	}

	p, err := m.pencapaian.Upsert(ctx, PencapaianInput{
		StudentID:    m.studentID,
		MateriAjarID: materiID,
		Status:       status,
		NilaiAngka:   nilai,
		Tanggal:      tanggal,
	})
	if err != nil {
		return "", false, err
	}
	return p.ID, existing == nil, nil
}

// pickChangedCell returns the filled month cell that differs from the
// existing record — the latest month wins when several differ. No cell
// means no change.
func pickChangedCell(existing *Pencapaian, cells map[int]int) (int, int, bool) {
	if len(cells) == 0 {
		return 0, 0, false
	}
	exMo := 0
	if existing != nil && existing.Tanggal != nil {
		if t, err := time.Parse("2006-01-02", *existing.Tanggal); err == nil {
			exMo = int(t.Month())
		}
	}
	bestMo, bestV, bestOK := 0, 0, false
	for mo, v := range cells {
		if existing != nil && mo == exMo && existing.NilaiAngka != nil && *existing.NilaiAngka == v {
			continue
		}
		if !bestOK || mo > bestMo {
			bestMo, bestV, bestOK = mo, v, true
		}
	}
	return bestMo, bestV, bestOK
}
