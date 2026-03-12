package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"music-server/internal/domain"
)

type ArtistRepository struct {
	db *sql.DB
}

var _ domain.ArtistRepository = (*ArtistRepository)(nil)

func NewArtistRepository(db *sql.DB) *ArtistRepository {
	return &ArtistRepository{db: db}
}

func (r *ArtistRepository) List(ctx context.Context) ([]domain.Artist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, cover_path, created_at, updated_at
		FROM artists
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list artists: %w", err)
	}
	defer rows.Close()

	artists := make([]domain.Artist, 0)
	for rows.Next() {
		var artist domain.Artist
		if err := rows.Scan(&artist.ID, &artist.Name, &artist.CoverPath, &artist.CreatedAt, &artist.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan artist: %w", err)
		}
		artists = append(artists, artist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artists rows: %w", err)
	}

	return artists, nil
}

func (r *ArtistRepository) GetByID(ctx context.Context, id string) (domain.Artist, error) {
	var artist domain.Artist
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, cover_path, created_at, updated_at
		FROM artists
		WHERE id = $1
	`, id).Scan(&artist.ID, &artist.Name, &artist.CoverPath, &artist.CreatedAt, &artist.UpdatedAt)
	if err != nil {
		return domain.Artist{}, fmt.Errorf("get artist by id: %w", err)
	}
	return artist, nil
}

func (r *ArtistRepository) GetByName(ctx context.Context, name string) (domain.Artist, error) {
	var artist domain.Artist
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, cover_path, created_at, updated_at
		FROM artists
		WHERE LOWER(name) = LOWER($1)
	`, name).Scan(&artist.ID, &artist.Name, &artist.CoverPath, &artist.CreatedAt, &artist.UpdatedAt)
	if err != nil {
		return domain.Artist{}, fmt.Errorf("get artist by name: %w", err)
	}
	return artist, nil
}

func (r *ArtistRepository) Search(ctx context.Context, query string) ([]domain.Artist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, cover_path, created_at, updated_at
		FROM artists
		WHERE name ILIKE '%' || $1 || '%'
		ORDER BY name ASC
	`, query)
	if err != nil {
		return nil, fmt.Errorf("search artists: %w", err)
	}
	defer rows.Close()

	artists := make([]domain.Artist, 0)
	for rows.Next() {
		var artist domain.Artist
		if err := rows.Scan(&artist.ID, &artist.Name, &artist.CoverPath, &artist.CreatedAt, &artist.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan artist: %w", err)
		}
		artists = append(artists, artist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artists rows: %w", err)
	}

	return artists, nil
}

func (r *ArtistRepository) Upsert(ctx context.Context, artist domain.Artist) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO artists (id, name, cover_path, created_at, updated_at)
		VALUES (
			COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()),
			$2, $3,
			COALESCE($4, NOW()),
			NOW()
		)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
			cover_path = EXCLUDED.cover_path,
			updated_at = NOW()
	`, artist.ID, artist.Name, artist.CoverPath, artist.CreatedAt)
	if err != nil {
		return fmt.Errorf("upsert artist: %w", err)
	}
	return nil
}
