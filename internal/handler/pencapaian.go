package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/fadhilkurnia/ppg-dashboard/internal/auth"
	"github.com/fadhilkurnia/ppg-dashboard/internal/httpx"
	"github.com/fadhilkurnia/ppg-dashboard/internal/store"
)

type Pencapaian struct {
	p         *store.PencapaianStore
	validator *validator.Validate
}

func NewPencapaian(p *store.PencapaianStore) *Pencapaian {
	return &Pencapaian{p: p, validator: validator.New()}
}

type pencapaianBody struct {
	StudentID    string  `json:"studentId"    validate:"required,min=1"`
	MateriAjarID string  `json:"materiAjarId" validate:"required,min=1"`
	Status       string  `json:"status"       validate:"required,oneof=belum proses tuntas"`
	NilaiAngka   *int    `json:"nilaiAngka,omitempty" validate:"omitempty,min=0,max=100"`
	Tanggal      *string `json:"tanggal,omitempty" validate:"omitempty,datetime=2006-01-02"`
	Catatan      *string `json:"catatan,omitempty" validate:"omitempty,max=2000"`
}

// List returns one student's pencapaian across ALL tingkat (all-ages view).
func (h *Pencapaian) List(w http.ResponseWriter, r *http.Request) {
	studentID := r.URL.Query().Get("studentId")
	if studentID == "" {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "studentId wajib diisi")
		return
	}
	rows, err := h.p.ListByStudent(r.Context(), studentID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal", "Gagal mengambil pencapaian")
		return
	}
	httpx.JSON(w, http.StatusOK, rows)
}

// ClassReport returns the Khayri-style per-tingkat semester report.
func (h *Pencapaian) ClassReport(w http.ResponseWriter, r *http.Request) {
	tingkatID := r.URL.Query().Get("tingkatId")
	semester, _ := strconv.Atoi(r.URL.Query().Get("semester"))
	if tingkatID == "" || semester == 0 {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "tingkatId dan semester wajib diisi")
		return
	}
	rows, err := h.p.ClassReport(r.Context(), tingkatID, semester)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal", "Gagal mengambil laporan")
		return
	}
	httpx.JSON(w, http.StatusOK, rows)
}

func (h *Pencapaian) Upsert(w http.ResponseWriter, r *http.Request) {
	if !canManageAchievement(r) {
		httpx.Error(w, http.StatusForbidden, "forbidden", "Akses tidak diizinkan")
		return
	}
	var b pencapaianBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "JSON tidak valid")
		return
	}
	if err := h.validator.Struct(b); err != nil {
		httpx.Error(w, http.StatusUnprocessableEntity, "validation", err.Error())
		return
	}
	c, _ := auth.ClaimsFrom(r.Context())
	p, err := h.p.Upsert(r.Context(), store.PencapaianInput{
		StudentID:    b.StudentID,
		MateriAjarID: b.MateriAjarID,
		Status:       b.Status,
		NilaiAngka:   b.NilaiAngka,
		Tanggal:      b.Tanggal,
		Catatan:      trimPtr(b.Catatan),
		RecordedBy:   c.UserID,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal", "Gagal menyimpan pencapaian")
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Pencapaian) Delete(w http.ResponseWriter, r *http.Request) {
	if !canManageAchievement(r) {
		httpx.Error(w, http.StatusForbidden, "forbidden", "Akses tidak diizinkan")
		return
	}
	err := h.p.Delete(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, store.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "not_found", "Pencapaian tidak ditemukan")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal", "Gagal menghapus pencapaian")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
