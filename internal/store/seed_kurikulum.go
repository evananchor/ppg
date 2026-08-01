package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed seed-data/kurikulum.sql
var kurikulumSeedSQL string

// SeedKurikulum loads the bundled curriculum (13 tingkat + 4700 materi_ajar
// rows exported from Tabel_Materi_per_Jenjang_Umur_per_Semester(1).xlsx)
// into a fresh database. Idempotent — skipped when tingkat already has rows.
func SeedKurikulum(ctx context.Context, db *sql.DB) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tingkat`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count tingkat: %w", err)
	}
	if n > 0 {
		return n, nil
	}

	// Split on the end-of-statement marker `);\n` because some materi values
	// contain literal newlines, so a per-line split would tear statements.
	cleaned := ""
	for _, line := range strings.Split(kurikulumSeedSQL, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			cleaned += line + "\n"
		}
	}
	chunks := strings.Split(cleaned, ");\n")
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for i, chunk := range chunks {
		stmt := strings.TrimSpace(chunk)
		if stmt == "" {
			continue
		}
		if !strings.HasSuffix(stmt, ");") {
			stmt += ");"
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return 0, fmt.Errorf("seed kurikulum stmt %d: %w", i+1, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return n, nil
}
