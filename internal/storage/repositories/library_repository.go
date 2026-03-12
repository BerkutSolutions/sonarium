package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"music-server/internal/domain"
)

type LibraryRepository struct {
	db *sql.DB
}

var _ domain.LibraryRepository = (*LibraryRepository)(nil)

func NewLibraryRepository(db *sql.DB) *LibraryRepository {
	return &LibraryRepository{db: db}
}

func (r *LibraryRepository) Get(ctx context.Context) (domain.Library, error) {
	var library domain.Library
	var lastScanAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT root_path, last_scan_at
		FROM library
		WHERE id = TRUE
	`).Scan(&library.RootPath, &lastScanAt)
	if err != nil {
		return domain.Library{}, fmt.Errorf("get library: %w", err)
	}
	if lastScanAt.Valid {
		library.LastScanAt = lastScanAt.Time
	}

	return library, nil
}

func (r *LibraryRepository) Save(ctx context.Context, library domain.Library) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO library (id, root_path, last_scan_at)
		VALUES (TRUE, $1, $2)
		ON CONFLICT (id) DO UPDATE
		SET root_path = EXCLUDED.root_path,
			last_scan_at = EXCLUDED.last_scan_at
	`, library.RootPath, library.LastScanAt)
	if err != nil {
		return fmt.Errorf("save library: %w", err)
	}

	return nil
}

func (r *LibraryRepository) UpdateLastScanAt(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO library (id, root_path, last_scan_at)
		VALUES (TRUE, '/music', NOW())
		ON CONFLICT (id) DO UPDATE
		SET last_scan_at = NOW()
	`)
	if err != nil {
		return fmt.Errorf("update library last_scan_at: %w", err)
	}

	return nil
}
