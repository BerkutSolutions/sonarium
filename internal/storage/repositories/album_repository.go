package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"music-server/internal/domain"
)

type AlbumRepository struct {
	db *sql.DB
}

var _ domain.AlbumRepository = (*AlbumRepository)(nil)

func NewAlbumRepository(db *sql.DB) *AlbumRepository {
	return &AlbumRepository{db: db}
}

func (r *AlbumRepository) List(ctx context.Context) ([]domain.Album, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, artist_id, year, cover_path, created_at, updated_at
		FROM albums
		WHERE EXISTS (
			SELECT 1
			FROM tracks t
			WHERE t.album_id = albums.id
		)
		ORDER BY title ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list albums: %w", err)
	}
	defer rows.Close()

	albums := make([]domain.Album, 0)
	for rows.Next() {
		var album domain.Album
		if err := rows.Scan(
			&album.ID,
			&album.Title,
			&album.ArtistID,
			&album.Year,
			&album.CoverPath,
			&album.CreatedAt,
			&album.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan album: %w", err)
		}
		albums = append(albums, album)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate albums rows: %w", err)
	}

	return albums, nil
}

func (r *AlbumRepository) GetByID(ctx context.Context, id string) (domain.Album, error) {
	var album domain.Album
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, artist_id, year, cover_path, created_at, updated_at
		FROM albums
		WHERE id = $1
	`, id).Scan(
		&album.ID,
		&album.Title,
		&album.ArtistID,
		&album.Year,
		&album.CoverPath,
		&album.CreatedAt,
		&album.UpdatedAt,
	)
	if err != nil {
		return domain.Album{}, fmt.Errorf("get album by id: %w", err)
	}

	return album, nil
}

func (r *AlbumRepository) GetByTitleAndArtistID(ctx context.Context, title, artistID string) (domain.Album, error) {
	var album domain.Album
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, artist_id, year, cover_path, created_at, updated_at
		FROM albums
		WHERE LOWER(BTRIM(title)) = LOWER(BTRIM($1))
		  AND artist_id = $2
	`, title, artistID).Scan(
		&album.ID,
		&album.Title,
		&album.ArtistID,
		&album.Year,
		&album.CoverPath,
		&album.CreatedAt,
		&album.UpdatedAt,
	)
	if err != nil {
		return domain.Album{}, fmt.Errorf("get album by title and artist: %w", err)
	}
	return album, nil
}

func (r *AlbumRepository) ListByArtistID(ctx context.Context, artistID string) ([]domain.Album, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, artist_id, year, cover_path, created_at, updated_at
		FROM albums
		WHERE artist_id = $1
		  AND EXISTS (
			SELECT 1
			FROM tracks t
			WHERE t.album_id = albums.id
		  )
		ORDER BY year ASC, title ASC
	`, artistID)
	if err != nil {
		return nil, fmt.Errorf("list albums by artist: %w", err)
	}
	defer rows.Close()

	albums := make([]domain.Album, 0)
	for rows.Next() {
		var album domain.Album
		if err := rows.Scan(
			&album.ID,
			&album.Title,
			&album.ArtistID,
			&album.Year,
			&album.CoverPath,
			&album.CreatedAt,
			&album.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan album: %w", err)
		}
		albums = append(albums, album)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate albums rows: %w", err)
	}

	return albums, nil
}

func (r *AlbumRepository) Search(ctx context.Context, query string) ([]domain.Album, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, artist_id, year, cover_path, created_at, updated_at
		FROM albums
		WHERE title ILIKE '%' || $1 || '%'
		  AND EXISTS (
			SELECT 1
			FROM tracks t
			WHERE t.album_id = albums.id
		  )
		ORDER BY title ASC
	`, query)
	if err != nil {
		return nil, fmt.Errorf("search albums: %w", err)
	}
	defer rows.Close()

	albums := make([]domain.Album, 0)
	for rows.Next() {
		var album domain.Album
		if err := rows.Scan(
			&album.ID,
			&album.Title,
			&album.ArtistID,
			&album.Year,
			&album.CoverPath,
			&album.CreatedAt,
			&album.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan album: %w", err)
		}
		albums = append(albums, album)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate albums rows: %w", err)
	}

	return albums, nil
}

func (r *AlbumRepository) Upsert(ctx context.Context, album domain.Album) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO albums (id, title, artist_id, year, cover_path, created_at, updated_at)
		VALUES (
			COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()),
			$2, $3, $4, $5,
			COALESCE($6, NOW()),
			NOW()
		)
		ON CONFLICT (id) DO UPDATE
		SET title = EXCLUDED.title,
			artist_id = EXCLUDED.artist_id,
			year = EXCLUDED.year,
			cover_path = EXCLUDED.cover_path,
			updated_at = NOW()
	`, album.ID, album.Title, album.ArtistID, album.Year, album.CoverPath, album.CreatedAt)
	if err != nil {
		return fmt.Errorf("upsert album: %w", err)
	}

	return nil
}
