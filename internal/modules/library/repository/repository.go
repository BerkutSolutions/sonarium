package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

type AlbumCard struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	ArtistID   string    `json:"artist_id"`
	ArtistName string    `json:"artist_name"`
	CoverPath  string    `json:"cover_path"`
	Year       int       `json:"year"`
	CreatedAt  time.Time `json:"created_at"`
}

type TrackCard struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	ArtistID        string    `json:"artist_id"`
	ArtistName      string    `json:"artist_name"`
	AlbumID         string    `json:"album_id"`
	AlbumTitle      string    `json:"album_title"`
	DurationSeconds int       `json:"duration_seconds"`
	PlayedAt        time.Time `json:"played_at,omitempty"`
	PositionSeconds int       `json:"position_seconds,omitempty"`
	ContextType     string    `json:"context_type,omitempty"`
	ContextID       string    `json:"context_id,omitempty"`
}

type Favorites struct {
	Albums  []AlbumCard  `json:"albums"`
	Artists []ArtistCard `json:"artists"`
	Tracks  []TrackCard  `json:"tracks"`
}

type ArtistCard struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CoverPath string `json:"cover_path"`
}

type HomeData struct {
	RecentAlbums      []AlbumCard `json:"recent_albums"`
	RecentTracks      []TrackCard `json:"recent_tracks"`
	ContinueListening []TrackCard `json:"continue_listening"`
	RandomAlbums      []AlbumCard `json:"random_albums"`
	Favorites         Favorites   `json:"favorites"`
}

type ArtistAlbumCount struct {
	ArtistID   string `json:"artist_id"`
	AlbumCount int    `json:"album_count"`
}

type PlayEvent struct {
	TrackID         string
	PositionSeconds int
	ContextType     string
	ContextID       string
}

type TrackUpdateInput struct {
	Title    string
	AlbumID  string
	ArtistID string
}

type CreateAlbumInput struct {
	Title    string
	ArtistID string
	Year     int
}

type AlbumUpdateInput struct {
	Title    string
	ArtistID string
	Year     int
}

type AlbumMergeInput struct {
	TargetAlbumID string
}

type ArtistUpdateInput struct {
	Name              string
	MergeIntoArtistID string
}

type StorageUserUsage struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	TrackCount  int    `json:"track_count"`
	BytesUsed   int64  `json:"bytes_used"`
}

type StorageUsage struct {
	TotalBytes int64              `json:"total_bytes"`
	Users      []StorageUserUsage `json:"users"`
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UploadConcurrency(ctx context.Context) (int, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `
		SELECT value
		FROM app_settings
		WHERE key = 'upload_concurrency'
	`).Scan(&raw)
	if err != nil {
		if err == sql.ErrNoRows {
			return 4, nil
		}
		return 0, fmt.Errorf("query upload concurrency: %w", err)
	}
	value, convErr := strconv.Atoi(raw)
	if convErr != nil {
		return 4, nil
	}
	if value < 1 {
		return 1, nil
	}
	if value > 10 {
		return 10, nil
	}
	return value, nil
}

func (r *Repository) SetUploadConcurrency(ctx context.Context, value int) error {
	if value < 1 {
		value = 1
	}
	if value > 10 {
		value = 10
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ('upload_concurrency', $1, NOW())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = NOW()
	`, strconv.Itoa(value))
	if err != nil {
		return fmt.Errorf("save upload concurrency: %w", err)
	}
	return nil
}

func (r *Repository) AutoCheckUpdates(ctx context.Context) (bool, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `
		SELECT value
		FROM app_settings
		WHERE key = 'auto_check_updates'
	`).Scan(&raw)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, fmt.Errorf("query auto check updates: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "1", "true", "yes", "on":
		return true, nil
	default:
		return false, nil
	}
}

func (r *Repository) SetAutoCheckUpdates(ctx context.Context, value bool) error {
	raw := "false"
	if value {
		raw = "true"
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ('auto_check_updates', $1, NOW())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = NOW()
	`, raw)
	if err != nil {
		return fmt.Errorf("save auto check updates: %w", err)
	}
	return nil
}

func (r *Repository) StorageUsage(ctx context.Context) (StorageUsage, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(file_size_bytes), 0)
		FROM tracks
	`).Scan(&total); err != nil {
		return StorageUsage{}, fmt.Errorf("query total storage: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			COALESCE(t.uploaded_by_user_id, 'system') AS user_id,
			COALESCE(NULLIF(u.display_name, ''), NULLIF(u.username, ''), 'System') AS display_name,
			COUNT(*) AS track_count,
			COALESCE(SUM(t.file_size_bytes), 0) AS bytes_used
		FROM tracks t
		LEFT JOIN app_users u ON u.id = t.uploaded_by_user_id
		GROUP BY COALESCE(t.uploaded_by_user_id, 'system'), COALESCE(NULLIF(u.display_name, ''), NULLIF(u.username, ''), 'System')
		ORDER BY bytes_used DESC, display_name ASC
	`)
	if err != nil {
		return StorageUsage{}, fmt.Errorf("query per-user storage: %w", err)
	}
	defer rows.Close()

	users := make([]StorageUserUsage, 0)
	for rows.Next() {
		var item StorageUserUsage
		if err := rows.Scan(&item.UserID, &item.DisplayName, &item.TrackCount, &item.BytesUsed); err != nil {
			return StorageUsage{}, fmt.Errorf("scan storage usage: %w", err)
		}
		users = append(users, item)
	}
	if err := rows.Err(); err != nil {
		return StorageUsage{}, fmt.Errorf("iterate storage usage: %w", err)
	}

	return StorageUsage{TotalBytes: total, Users: users}, nil
}

func (r *Repository) ClearLibrary(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin clear library tx: %w", err)
	}
	defer tx.Rollback()

	statements := []string{
		`DELETE FROM entity_shares WHERE entity_type IN ('album', 'artist', 'track')`,
		`DELETE FROM play_history`,
		`DELETE FROM user_favorite_tracks`,
		`DELETE FROM user_favorite_albums`,
		`DELETE FROM user_favorite_artists`,
		`DELETE FROM playlist_tracks`,
		`DELETE FROM tracks`,
		`DELETE FROM albums`,
		`DELETE FROM artists`,
		`DELETE FROM library_file_fingerprints`,
		`DELETE FROM library_album_stats`,
		`DELETE FROM library_artist_stats`,
		`UPDATE library SET last_scan_at = NULL WHERE id = TRUE`,
	}
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("clear library: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit clear library tx: %w", err)
	}
	return nil
}

func (r *Repository) RecordPlayEvent(ctx context.Context, userID string, event PlayEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO play_history (user_id, track_id, played_at, position_seconds, context_type, context_id)
		VALUES ($1, $2, NOW(), $3, $4, $5)
	`, userID, event.TrackID, event.PositionSeconds, event.ContextType, event.ContextID)
	if err != nil {
		return fmt.Errorf("insert play history: %w", err)
	}
	return nil
}

func (r *Repository) ToggleFavoriteTrack(ctx context.Context, userID, trackID string) (bool, error) {
	return toggle(ctx, r.db, `
		DELETE FROM user_favorite_tracks WHERE user_id = $1 AND track_id = $2
	`, `
		INSERT INTO user_favorite_tracks (user_id, track_id) VALUES ($1, $2)
	`, userID, trackID)
}

func (r *Repository) ToggleFavoriteAlbum(ctx context.Context, userID, albumID string) (bool, error) {
	return toggle(ctx, r.db, `
		DELETE FROM user_favorite_albums WHERE user_id = $1 AND album_id = $2
	`, `
		INSERT INTO user_favorite_albums (user_id, album_id) VALUES ($1, $2)
	`, userID, albumID)
}

func (r *Repository) ToggleFavoriteArtist(ctx context.Context, userID, artistID string) (bool, error) {
	return toggle(ctx, r.db, `
		DELETE FROM user_favorite_artists WHERE user_id = $1 AND artist_id = $2
	`, `
		INSERT INTO user_favorite_artists (user_id, artist_id) VALUES ($1, $2)
	`, userID, artistID)
}

func (r *Repository) RandomAlbums(ctx context.Context, limit int) ([]AlbumCard, error) {
	if limit <= 0 {
		limit = 12
	}
	seed := rand.Float64()
	head, err := r.queryAlbums(ctx, `
		SELECT a.id, a.title, a.artist_id, ar.name, a.cover_path, a.year, a.created_at
		FROM albums a
		JOIN artists ar ON ar.id = a.artist_id
		WHERE a.random_key >= $1
		ORDER BY a.random_key ASC
		LIMIT $2
	`, seed, limit)
	if err != nil {
		return nil, err
	}
	if len(head) >= limit {
		return head, nil
	}
	tail, err := r.queryAlbums(ctx, `
		SELECT a.id, a.title, a.artist_id, ar.name, a.cover_path, a.year, a.created_at
		FROM albums a
		JOIN artists ar ON ar.id = a.artist_id
		WHERE a.random_key < $1
		ORDER BY a.random_key ASC
		LIMIT $2
	`, seed, limit-len(head))
	if err != nil {
		return nil, err
	}
	return append(head, tail...), nil
}

func (r *Repository) HomeData(ctx context.Context, userID string, limit int) (HomeData, error) {
	if limit <= 0 {
		limit = 12
	}
	recentAlbums, err := r.queryAlbums(ctx, `
		SELECT a.id, a.title, a.artist_id, ar.name, a.cover_path, a.year, a.created_at
		FROM albums a
		JOIN artists ar ON ar.id = a.artist_id
		ORDER BY a.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return HomeData{}, err
	}
	recentTracks, err := r.queryTracks(ctx, `
		WITH last_play AS (
			SELECT DISTINCT ON (ph.track_id)
				ph.track_id, ph.played_at, ph.position_seconds, ph.context_type, ph.context_id
			FROM play_history ph
			WHERE ph.user_id = $1
			ORDER BY ph.track_id, ph.played_at DESC
		)
		SELECT t.id, t.title, t.artist_id, ar.name, t.album_id, al.title, t.duration_seconds,
		       lp.played_at, lp.position_seconds, lp.context_type, lp.context_id
		FROM last_play lp
		JOIN tracks t ON t.id = lp.track_id
		JOIN artists ar ON ar.id = t.artist_id
		JOIN albums al ON al.id = t.album_id
		ORDER BY lp.played_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return HomeData{}, err
	}
	continueListening, err := r.queryTracks(ctx, `
		WITH last_play AS (
			SELECT DISTINCT ON (ph.track_id)
				ph.track_id, ph.played_at, ph.position_seconds, ph.context_type, ph.context_id
			FROM play_history ph
			WHERE ph.user_id = $1
			ORDER BY ph.track_id, ph.played_at DESC
		)
		SELECT t.id, t.title, t.artist_id, ar.name, t.album_id, al.title, t.duration_seconds,
		       lp.played_at, lp.position_seconds, lp.context_type, lp.context_id
		FROM last_play lp
		JOIN tracks t ON t.id = lp.track_id
		JOIN artists ar ON ar.id = t.artist_id
		JOIN albums al ON al.id = t.album_id
		WHERE lp.position_seconds > 30 AND lp.position_seconds < GREATEST(t.duration_seconds - 15, 31)
		ORDER BY lp.played_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return HomeData{}, err
	}
	randomAlbums, err := r.RandomAlbums(ctx, limit)
	if err != nil {
		return HomeData{}, err
	}
	favorites, err := r.Favorites(ctx, userID, limit)
	if err != nil {
		return HomeData{}, err
	}

	return HomeData{
		RecentAlbums:      recentAlbums,
		RecentTracks:      recentTracks,
		ContinueListening: continueListening,
		RandomAlbums:      randomAlbums,
		Favorites:         favorites,
	}, nil
}

func (r *Repository) Favorites(ctx context.Context, userID string, limit int) (Favorites, error) {
	albums, err := r.queryAlbums(ctx, `
		SELECT a.id, a.title, a.artist_id, ar.name, a.cover_path, a.year, a.created_at
		FROM user_favorite_albums uf
		JOIN albums a ON a.id = uf.album_id
		JOIN artists ar ON ar.id = a.artist_id
		WHERE uf.user_id = $1
		ORDER BY uf.created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return Favorites{}, err
	}
	artists, err := r.queryArtists(ctx, `
		SELECT ar.id, ar.name, ar.cover_path
		FROM user_favorite_artists uf
		JOIN artists ar ON ar.id = uf.artist_id
		WHERE uf.user_id = $1
		ORDER BY uf.created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return Favorites{}, err
	}
	tracks, err := r.queryTracks(ctx, `
		SELECT t.id, t.title, t.artist_id, ar.name, t.album_id, al.title, t.duration_seconds,
		       NOW(), 0, '', ''
		FROM user_favorite_tracks uf
		JOIN tracks t ON t.id = uf.track_id
		JOIN artists ar ON ar.id = t.artist_id
		JOIN albums al ON al.id = t.album_id
		WHERE uf.user_id = $1
		ORDER BY uf.created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return Favorites{}, err
	}
	return Favorites{Albums: albums, Artists: artists, Tracks: tracks}, nil
}

func (r *Repository) RefreshAggregates(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO library_album_stats (album_id, tracks_count, total_duration_seconds, updated_at)
		SELECT t.album_id, COUNT(*), COALESCE(SUM(t.duration_seconds), 0), NOW()
		FROM tracks t
		GROUP BY t.album_id
		ON CONFLICT (album_id) DO UPDATE
		SET tracks_count = EXCLUDED.tracks_count,
		    total_duration_seconds = EXCLUDED.total_duration_seconds,
		    updated_at = NOW();

		DELETE FROM library_album_stats s
		WHERE NOT EXISTS (SELECT 1 FROM albums a WHERE a.id = s.album_id);

		INSERT INTO library_artist_stats (artist_id, albums_count, tracks_count, total_duration_seconds, updated_at)
		SELECT ar.id,
		       COUNT(DISTINCT al.id),
		       COUNT(t.id),
		       COALESCE(SUM(t.duration_seconds), 0),
		       NOW()
		FROM artists ar
		LEFT JOIN albums al ON al.artist_id = ar.id
		LEFT JOIN tracks t ON t.artist_id = ar.id
		GROUP BY ar.id
		ON CONFLICT (artist_id) DO UPDATE
		SET albums_count = EXCLUDED.albums_count,
		    tracks_count = EXCLUDED.tracks_count,
		    total_duration_seconds = EXCLUDED.total_duration_seconds,
		    updated_at = NOW();

		DELETE FROM library_artist_stats s
		WHERE NOT EXISTS (SELECT 1 FROM artists a WHERE a.id = s.artist_id);
	`)
	if err != nil {
		return fmt.Errorf("refresh aggregates: %w", err)
	}
	return nil
}

func (r *Repository) ArtistAlbumCounts(ctx context.Context) ([]ArtistAlbumCount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT artist_id, albums_count
		FROM library_artist_stats
	`)
	if err != nil {
		return nil, fmt.Errorf("query artist album counts: %w", err)
	}
	defer rows.Close()
	items := make([]ArtistAlbumCount, 0)
	for rows.Next() {
		var item ArtistAlbumCount
		if err := rows.Scan(&item.ArtistID, &item.AlbumCount); err != nil {
			return nil, fmt.Errorf("scan artist album count: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artist album counts: %w", err)
	}
	return items, nil
}

func (r *Repository) DeleteTrack(ctx context.Context, trackID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tracks WHERE id = $1`, trackID)
	if err != nil {
		return fmt.Errorf("delete track: %w", err)
	}
	return nil
}

func (r *Repository) RenameTrack(ctx context.Context, trackID, title string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tracks
		SET title = $2, updated_at = NOW()
		WHERE id = $1
	`, trackID, title)
	if err != nil {
		return fmt.Errorf("rename track: %w", err)
	}
	return nil
}

func (r *Repository) DeleteAlbum(ctx context.Context, albumID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM albums WHERE id = $1`, albumID)
	if err != nil {
		return fmt.Errorf("delete album: %w", err)
	}
	return nil
}

func (r *Repository) RenameAlbum(ctx context.Context, albumID, title string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE albums
		SET title = $2, updated_at = NOW()
		WHERE id = $1
	`, albumID, title)
	if err != nil {
		return fmt.Errorf("rename album: %w", err)
	}
	return nil
}

func (r *Repository) UpdateTrack(ctx context.Context, trackID string, input TrackUpdateInput) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tracks
		SET title = $2, album_id = $3, artist_id = $4, updated_at = NOW()
		WHERE id = $1
	`, trackID, input.Title, input.AlbumID, input.ArtistID)
	if err != nil {
		return fmt.Errorf("update track: %w", err)
	}
	return nil
}

func (r *Repository) CreateAlbum(ctx context.Context, input CreateAlbumInput) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO albums (id, title, artist_id, year, cover_path, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, '', NOW(), NOW())
		RETURNING id
	`, input.Title, input.ArtistID, input.Year).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create album: %w", err)
	}
	return id, nil
}

func (r *Repository) UpdateAlbum(ctx context.Context, albumID string, input AlbumUpdateInput) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE albums
		SET title = $2, artist_id = $3, year = $4, updated_at = NOW()
		WHERE id = $1
	`, albumID, input.Title, input.ArtistID, input.Year)
	if err != nil {
		return fmt.Errorf("update album: %w", err)
	}
	return nil
}

func (r *Repository) MergeAlbum(ctx context.Context, albumID string, input AlbumMergeInput) error {
	targetAlbumID := input.TargetAlbumID
	if targetAlbumID == "" || targetAlbumID == albumID {
		return fmt.Errorf("target album id is required")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin merge album tx: %w", err)
	}
	defer tx.Rollback()

	type albumInfo struct {
		ID        string
		ArtistID  string
		Title     string
		Year      int
		CoverPath string
	}

	var source albumInfo
	if err := tx.QueryRowContext(ctx, `
		SELECT id, artist_id, title, COALESCE(year, 0), COALESCE(cover_path, '')
		FROM albums
		WHERE id = $1
	`, albumID).Scan(&source.ID, &source.ArtistID, &source.Title, &source.Year, &source.CoverPath); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("source album not found")
		}
		return fmt.Errorf("load source album: %w", err)
	}

	var target albumInfo
	if err := tx.QueryRowContext(ctx, `
		SELECT id, artist_id, title, COALESCE(year, 0), COALESCE(cover_path, '')
		FROM albums
		WHERE id = $1
	`, targetAlbumID).Scan(&target.ID, &target.ArtistID, &target.Title, &target.Year, &target.CoverPath); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("target album not found")
		}
		return fmt.Errorf("load target album: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE tracks
		SET album_id = $2, updated_at = NOW()
		WHERE album_id = $1
	`, source.ID, target.ID); err != nil {
		return fmt.Errorf("reassign album tracks: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_favorite_albums (user_id, album_id, created_at)
		SELECT ufa.user_id, $2, ufa.created_at
		FROM user_favorite_albums ufa
		WHERE ufa.album_id = $1
		ON CONFLICT (user_id, album_id) DO NOTHING
	`, source.ID, target.ID); err != nil {
		return fmt.Errorf("merge album favorites: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM user_favorite_albums
		WHERE album_id = $1
	`, source.ID); err != nil {
		return fmt.Errorf("delete source album favorites: %w", err)
	}

	if source.CoverPath != "" && target.CoverPath == "" {
		if _, err := tx.ExecContext(ctx, `
			UPDATE albums
			SET cover_path = $2, updated_at = NOW()
			WHERE id = $1
		`, target.ID, source.CoverPath); err != nil {
			return fmt.Errorf("copy album cover: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM albums WHERE id = $1`, source.ID); err != nil {
		return fmt.Errorf("delete source album: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit merge album tx: %w", err)
	}
	return nil
}

func (r *Repository) UpdateArtist(ctx context.Context, artistID string, input ArtistUpdateInput) error {
	if input.MergeIntoArtistID != "" && input.MergeIntoArtistID != artistID {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin merge artist tx: %w", err)
		}
		defer tx.Rollback()

		var sourceCoverPath string
		if err := tx.QueryRowContext(ctx, `SELECT cover_path FROM artists WHERE id = $1`, artistID).Scan(&sourceCoverPath); err != nil {
			return fmt.Errorf("load source artist: %w", err)
		}

		var targetCoverPath string
		if err := tx.QueryRowContext(ctx, `SELECT cover_path FROM artists WHERE id = $1`, input.MergeIntoArtistID).Scan(&targetCoverPath); err != nil {
			return fmt.Errorf("load target artist: %w", err)
		}

		type mergeAlbum struct {
			ID        string
			Title     string
			Year      int
			CoverPath string
		}
		rows, err := tx.QueryContext(ctx, `
			SELECT id, title, year, cover_path
			FROM albums
			WHERE artist_id = $1
		`, artistID)
		if err != nil {
			return fmt.Errorf("load source artist albums: %w", err)
		}
		sourceAlbums := make([]mergeAlbum, 0)
		for rows.Next() {
			var album mergeAlbum
			if err := rows.Scan(&album.ID, &album.Title, &album.Year, &album.CoverPath); err != nil {
				rows.Close()
				return fmt.Errorf("scan source artist album: %w", err)
			}
			sourceAlbums = append(sourceAlbums, album)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("iterate source artist albums: %w", err)
		}
		rows.Close()

		for _, album := range sourceAlbums {
			var targetAlbumID string
			var targetAlbumCoverPath string
			err := tx.QueryRowContext(ctx, `
				SELECT id, COALESCE(cover_path, '')
				FROM albums
				WHERE artist_id = $1
				  AND id <> $4
				  AND LOWER(BTRIM(title)) = LOWER(BTRIM($2))
				  AND COALESCE(year, 0) = COALESCE($3, 0)
				ORDER BY created_at ASC
				LIMIT 1
			`, input.MergeIntoArtistID, album.Title, album.Year, album.ID).Scan(&targetAlbumID, &targetAlbumCoverPath)
			switch {
			case err == sql.ErrNoRows:
				if _, err := tx.ExecContext(ctx, `
					UPDATE albums
					SET artist_id = $2, updated_at = NOW()
					WHERE id = $1
				`, album.ID, input.MergeIntoArtistID); err != nil {
					return fmt.Errorf("reassign artist album: %w", err)
				}
				if _, err := tx.ExecContext(ctx, `
					UPDATE tracks
					SET artist_id = $2, updated_at = NOW()
					WHERE album_id = $1
				`, album.ID, input.MergeIntoArtistID); err != nil {
					return fmt.Errorf("reassign album tracks: %w", err)
				}
			case err != nil:
				return fmt.Errorf("find merge target album: %w", err)
			default:
				if _, err := tx.ExecContext(ctx, `
					UPDATE tracks
					SET artist_id = $2, album_id = $3, updated_at = NOW()
					WHERE album_id = $1
				`, album.ID, input.MergeIntoArtistID, targetAlbumID); err != nil {
					return fmt.Errorf("merge duplicate album tracks: %w", err)
				}

				if _, err := tx.ExecContext(ctx, `
					INSERT INTO user_favorite_albums (user_id, album_id, created_at)
					SELECT ufa.user_id, $2, ufa.created_at
					FROM user_favorite_albums ufa
					WHERE ufa.album_id = $1
					ON CONFLICT (user_id, album_id) DO NOTHING
				`, album.ID, targetAlbumID); err != nil {
					return fmt.Errorf("merge duplicate album favorites: %w", err)
				}

				if _, err := tx.ExecContext(ctx, `
					DELETE FROM user_favorite_albums
					WHERE album_id = $1
				`, album.ID); err != nil {
					return fmt.Errorf("delete duplicate album favorites: %w", err)
				}

				if album.CoverPath != "" && targetAlbumCoverPath == "" {
					if _, err := tx.ExecContext(ctx, `
						UPDATE albums
						SET cover_path = $2, updated_at = NOW()
						WHERE id = $1
					`, targetAlbumID, album.CoverPath); err != nil {
						return fmt.Errorf("copy duplicate album cover: %w", err)
					}
				}

				if _, err := tx.ExecContext(ctx, `DELETE FROM albums WHERE id = $1`, album.ID); err != nil {
					return fmt.Errorf("delete duplicate album: %w", err)
				}
			}
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE tracks
			SET artist_id = $2, updated_at = NOW()
			WHERE artist_id = $1
		`, artistID, input.MergeIntoArtistID); err != nil {
			return fmt.Errorf("merge artist tracks: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_favorite_artists (user_id, artist_id, created_at)
			SELECT ufa.user_id, $2, ufa.created_at
			FROM user_favorite_artists ufa
			WHERE ufa.artist_id = $1
			ON CONFLICT (user_id, artist_id) DO NOTHING
		`, artistID, input.MergeIntoArtistID); err != nil {
			return fmt.Errorf("merge artist favorites: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			DELETE FROM user_favorite_artists
			WHERE artist_id = $1
		`, artistID); err != nil {
			return fmt.Errorf("delete old artist favorites: %w", err)
		}

		if sourceCoverPath != "" && targetCoverPath == "" {
			if _, err := tx.ExecContext(ctx, `
				UPDATE artists
				SET cover_path = $2, updated_at = NOW()
				WHERE id = $1
			`, input.MergeIntoArtistID, sourceCoverPath); err != nil {
				return fmt.Errorf("copy artist cover: %w", err)
			}
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM artists WHERE id = $1`, artistID); err != nil {
			return fmt.Errorf("delete merged artist: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit merge artist tx: %w", err)
		}
		return nil
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE artists
		SET name = $2, updated_at = NOW()
		WHERE id = $1
	`, artistID, input.Name)
	if err != nil {
		return fmt.Errorf("update artist: %w", err)
	}
	return nil
}

func (r *Repository) DeleteArtist(ctx context.Context, artistID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM artists WHERE id = $1`, artistID)
	if err != nil {
		return fmt.Errorf("delete artist: %w", err)
	}
	return nil
}

func (r *Repository) queryAlbums(ctx context.Context, query string, args ...any) ([]AlbumCard, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query albums: %w", err)
	}
	defer rows.Close()
	items := make([]AlbumCard, 0)
	for rows.Next() {
		var item AlbumCard
		if err := rows.Scan(&item.ID, &item.Title, &item.ArtistID, &item.ArtistName, &item.CoverPath, &item.Year, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan album card: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate albums rows: %w", err)
	}
	return items, nil
}

func (r *Repository) queryArtists(ctx context.Context, query string, args ...any) ([]ArtistCard, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query artists: %w", err)
	}
	defer rows.Close()
	items := make([]ArtistCard, 0)
	for rows.Next() {
		var item ArtistCard
		if err := rows.Scan(&item.ID, &item.Name, &item.CoverPath); err != nil {
			return nil, fmt.Errorf("scan artist card: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artists rows: %w", err)
	}
	return items, nil
}

func (r *Repository) queryTracks(ctx context.Context, query string, args ...any) ([]TrackCard, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tracks: %w", err)
	}
	defer rows.Close()
	items := make([]TrackCard, 0)
	for rows.Next() {
		var item TrackCard
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.ArtistID,
			&item.ArtistName,
			&item.AlbumID,
			&item.AlbumTitle,
			&item.DurationSeconds,
			&item.PlayedAt,
			&item.PositionSeconds,
			&item.ContextType,
			&item.ContextID,
		); err != nil {
			return nil, fmt.Errorf("scan track card: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracks rows: %w", err)
	}
	return items, nil
}

func toggle(ctx context.Context, db *sql.DB, deleteQuery, insertQuery, userID, entityID string) (bool, error) {
	result, err := db.ExecContext(ctx, deleteQuery, userID, entityID)
	if err != nil {
		return false, fmt.Errorf("delete favorite: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("favorite rows affected: %w", err)
	}
	if affected > 0 {
		return false, nil
	}
	if _, err := db.ExecContext(ctx, insertQuery, userID, entityID); err != nil {
		return false, fmt.Errorf("insert favorite: %w", err)
	}
	return true, nil
}
