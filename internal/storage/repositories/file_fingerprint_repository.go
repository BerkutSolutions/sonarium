package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type FileFingerprint struct {
	FilePath        string
	FileSize        int64
	ModTime         time.Time
	FingerprintHash string
}

type FileFingerprintRepository struct {
	db *sql.DB
}

func NewFileFingerprintRepository(db *sql.DB) *FileFingerprintRepository {
	return &FileFingerprintRepository{db: db}
}

func (r *FileFingerprintRepository) List(ctx context.Context) ([]FileFingerprint, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT file_path, file_size, mod_time, fingerprint_hash
		FROM library_file_fingerprints
	`)
	if err != nil {
		return nil, fmt.Errorf("list fingerprints: %w", err)
	}
	defer rows.Close()

	items := make([]FileFingerprint, 0)
	for rows.Next() {
		var item FileFingerprint
		if err := rows.Scan(&item.FilePath, &item.FileSize, &item.ModTime, &item.FingerprintHash); err != nil {
			return nil, fmt.Errorf("scan fingerprint: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fingerprints rows: %w", err)
	}

	return items, nil
}

func (r *FileFingerprintRepository) Upsert(ctx context.Context, filePath string, fileSize int64, modTime time.Time, fingerprintHash string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO library_file_fingerprints (file_path, file_size, mod_time, fingerprint_hash, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (file_path) DO UPDATE
		SET file_size = EXCLUDED.file_size,
			mod_time = EXCLUDED.mod_time,
			fingerprint_hash = EXCLUDED.fingerprint_hash,
			updated_at = NOW()
	`, filePath, fileSize, modTime.UTC(), fingerprintHash)
	if err != nil {
		return fmt.Errorf("upsert fingerprint: %w", err)
	}
	return nil
}
