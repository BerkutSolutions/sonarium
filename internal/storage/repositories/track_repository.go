package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"music-server/internal/domain"
)

type TrackRepository struct {
	db *sql.DB
}

var _ domain.TrackRepository = (*TrackRepository)(nil)

func NewTrackRepository(db *sql.DB) *TrackRepository {
	return &TrackRepository{db: db}
}

func (r *TrackRepository) List(ctx context.Context) ([]domain.Track, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, album_id, artist_id, track_number, duration_seconds, file_path, genre, codec, bitrate, file_size_bytes, COALESCE(uploaded_by_user_id, ''), replay_gain_track, replay_gain_album, created_at, updated_at
		FROM tracks
		ORDER BY artist_id, album_id, track_number ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list tracks: %w", err)
	}
	defer rows.Close()

	tracks := make([]domain.Track, 0)
	for rows.Next() {
		track, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracks rows: %w", err)
	}

	return tracks, nil
}

func (r *TrackRepository) GetByID(ctx context.Context, id string) (domain.Track, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, album_id, artist_id, track_number, duration_seconds, file_path, genre, codec, bitrate, file_size_bytes, COALESCE(uploaded_by_user_id, ''), replay_gain_track, replay_gain_album, created_at, updated_at
		FROM tracks
		WHERE id = $1
	`, id)
	if err != nil {
		return domain.Track{}, fmt.Errorf("get track by id: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return domain.Track{}, sql.ErrNoRows
	}

	track, err := scanTrack(rows)
	if err != nil {
		return domain.Track{}, err
	}

	return track, nil
}

func (r *TrackRepository) GetByFilePath(ctx context.Context, filePath string) (domain.Track, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, album_id, artist_id, track_number, duration_seconds, file_path, genre, codec, bitrate, file_size_bytes, COALESCE(uploaded_by_user_id, ''), replay_gain_track, replay_gain_album, created_at, updated_at
		FROM tracks
		WHERE file_path = $1
	`, filePath)
	if err != nil {
		return domain.Track{}, fmt.Errorf("get track by file path: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return domain.Track{}, sql.ErrNoRows
	}

	track, err := scanTrack(rows)
	if err != nil {
		return domain.Track{}, err
	}
	return track, nil
}

func (r *TrackRepository) ListByAlbumID(ctx context.Context, albumID string) ([]domain.Track, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, album_id, artist_id, track_number, duration_seconds, file_path, genre, codec, bitrate, file_size_bytes, COALESCE(uploaded_by_user_id, ''), replay_gain_track, replay_gain_album, created_at, updated_at
		FROM tracks
		WHERE album_id = $1
		ORDER BY track_number ASC
	`, albumID)
	if err != nil {
		return nil, fmt.Errorf("list tracks by album: %w", err)
	}
	defer rows.Close()

	tracks := make([]domain.Track, 0)
	for rows.Next() {
		track, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracks rows: %w", err)
	}

	return tracks, nil
}

func (r *TrackRepository) ListByArtistID(ctx context.Context, artistID string) ([]domain.Track, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, album_id, artist_id, track_number, duration_seconds, file_path, genre, codec, bitrate, file_size_bytes, COALESCE(uploaded_by_user_id, ''), replay_gain_track, replay_gain_album, created_at, updated_at
		FROM tracks
		WHERE artist_id = $1
		ORDER BY album_id, track_number ASC
	`, artistID)
	if err != nil {
		return nil, fmt.Errorf("list tracks by artist: %w", err)
	}
	defer rows.Close()

	tracks := make([]domain.Track, 0)
	for rows.Next() {
		track, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracks rows: %w", err)
	}

	return tracks, nil
}

func (r *TrackRepository) Search(ctx context.Context, query string) ([]domain.Track, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, album_id, artist_id, track_number, duration_seconds, file_path, genre, codec, bitrate, file_size_bytes, COALESCE(uploaded_by_user_id, ''), replay_gain_track, replay_gain_album, created_at, updated_at
		FROM tracks
		WHERE title ILIKE '%' || $1 || '%'
		ORDER BY title ASC
	`, query)
	if err != nil {
		return nil, fmt.Errorf("search tracks: %w", err)
	}
	defer rows.Close()

	tracks := make([]domain.Track, 0)
	for rows.Next() {
		track, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracks rows: %w", err)
	}

	return tracks, nil
}

func (r *TrackRepository) Upsert(ctx context.Context, track domain.Track) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tracks (
			id, title, album_id, artist_id, track_number, duration_seconds, file_path, genre, codec, bitrate, file_size_bytes, uploaded_by_user_id, replay_gain_track, replay_gain_album, created_at, updated_at
		)
		VALUES (
			COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()),
			$2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NULLIF($12, ''), $13, $14,
			COALESCE($15, NOW()),
			NOW()
		)
		ON CONFLICT (id) DO UPDATE
		SET title = EXCLUDED.title,
			album_id = EXCLUDED.album_id,
			artist_id = EXCLUDED.artist_id,
			track_number = EXCLUDED.track_number,
			duration_seconds = EXCLUDED.duration_seconds,
			file_path = EXCLUDED.file_path,
			genre = EXCLUDED.genre,
			codec = EXCLUDED.codec,
			bitrate = EXCLUDED.bitrate,
			file_size_bytes = EXCLUDED.file_size_bytes,
			uploaded_by_user_id = EXCLUDED.uploaded_by_user_id,
			replay_gain_track = EXCLUDED.replay_gain_track,
			replay_gain_album = EXCLUDED.replay_gain_album,
			updated_at = NOW()
	`,
		track.ID,
		track.Title,
		track.AlbumID,
		track.ArtistID,
		track.TrackNumber,
		int64(track.Duration.Seconds()),
		track.FilePath,
		track.Genre,
		track.Codec,
		track.Bitrate,
		track.FileSizeBytes,
		track.UploadedByUserID,
		track.ReplayGainTrack,
		track.ReplayGainAlbum,
		track.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert track: %w", err)
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTrack(scanner rowScanner) (domain.Track, error) {
	var track domain.Track
	var durationSeconds int64
	if err := scanner.Scan(
		&track.ID,
		&track.Title,
		&track.AlbumID,
		&track.ArtistID,
		&track.TrackNumber,
		&durationSeconds,
		&track.FilePath,
		&track.Genre,
		&track.Codec,
		&track.Bitrate,
		&track.FileSizeBytes,
		&track.UploadedByUserID,
		&track.ReplayGainTrack,
		&track.ReplayGainAlbum,
		&track.CreatedAt,
		&track.UpdatedAt,
	); err != nil {
		return domain.Track{}, fmt.Errorf("scan track: %w", err)
	}
	track.Duration = time.Duration(durationSeconds) * time.Second
	return track, nil
}
