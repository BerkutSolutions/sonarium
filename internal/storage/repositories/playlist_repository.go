package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"music-server/internal/domain"
)

type PlaylistRepository struct {
	db *sql.DB
}

var _ domain.PlaylistRepository = (*PlaylistRepository)(nil)

func NewPlaylistRepository(db *sql.DB) *PlaylistRepository {
	return &PlaylistRepository{db: db}
}

func (r *PlaylistRepository) List(ctx context.Context) ([]domain.Playlist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, COALESCE(description, ''), COALESCE(owner_user_id, ''), created_at, updated_at, ''
		FROM playlists
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list playlists: %w", err)
	}
	defer rows.Close()

	playlists := make([]domain.Playlist, 0)
	for rows.Next() {
		var playlist domain.Playlist
		if err := rows.Scan(
			&playlist.ID,
			&playlist.Name,
			&playlist.Description,
			&playlist.OwnerUserID,
			&playlist.CreatedAt,
			&playlist.UpdatedAt,
			&playlist.AccessRole,
		); err != nil {
			return nil, fmt.Errorf("scan playlist: %w", err)
		}
		playlists = append(playlists, playlist)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playlists rows: %w", err)
	}
	return playlists, nil
}

func (r *PlaylistRepository) ListAccessible(ctx context.Context, userID string) ([]domain.Playlist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT
			p.id,
			p.name,
			COALESCE(p.description, ''),
			COALESCE(p.owner_user_id, ''),
			p.created_at,
			p.updated_at,
			CASE
				WHEN p.owner_user_id = $1 THEN 'owner'
				WHEN EXISTS (
					SELECT 1
					FROM entity_shares s
					WHERE s.entity_type = 'playlist'
					  AND s.entity_id = p.id::text
					  AND s.shared_with_user_id = $1
					  AND s.permission = 'editor'
				) THEN 'editor'
				WHEN EXISTS (
					SELECT 1
					FROM entity_shares s
					WHERE s.entity_type = 'playlist'
					  AND s.entity_id = p.id::text
					  AND s.shared_with_user_id = $1
				) THEN 'listener'
				ELSE ''
			END AS access_role
		FROM playlists p
		WHERE p.owner_user_id = $1
		   OR EXISTS (
				SELECT 1
				FROM entity_shares s
				WHERE s.entity_type = 'playlist'
				  AND s.entity_id = p.id::text
				  AND s.shared_with_user_id = $1
			)
		ORDER BY name ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list playlists: %w", err)
	}
	defer rows.Close()

	playlists := make([]domain.Playlist, 0)
	for rows.Next() {
		var playlist domain.Playlist
		if err := rows.Scan(
			&playlist.ID,
			&playlist.Name,
			&playlist.Description,
			&playlist.OwnerUserID,
			&playlist.CreatedAt,
			&playlist.UpdatedAt,
			&playlist.AccessRole,
		); err != nil {
			return nil, fmt.Errorf("scan playlist: %w", err)
		}
		playlists = append(playlists, playlist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playlists rows: %w", err)
	}

	return playlists, nil
}

func (r *PlaylistRepository) GetByID(ctx context.Context, id string) (domain.Playlist, error) {
	var playlist domain.Playlist
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(description, ''), COALESCE(owner_user_id, ''), created_at, updated_at, ''
		FROM playlists
		WHERE id = $1
	`, id).Scan(
		&playlist.ID,
		&playlist.Name,
		&playlist.Description,
		&playlist.OwnerUserID,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
		&playlist.AccessRole,
	)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("get playlist by id: %w", err)
	}
	return playlist, nil
}

func (r *PlaylistRepository) GetAccessibleByID(ctx context.Context, id, userID, shareToken string) (domain.Playlist, error) {
	var playlist domain.Playlist
	err := r.db.QueryRowContext(ctx, `
		SELECT
			p.id,
			p.name,
			COALESCE(p.description, ''),
			COALESCE(p.owner_user_id, ''),
			p.created_at,
			p.updated_at,
			CASE
				WHEN p.owner_user_id = $2 THEN 'owner'
				WHEN EXISTS (
					SELECT 1
					FROM entity_shares s
					WHERE s.entity_type = 'playlist'
					  AND s.entity_id = p.id::text
					  AND s.shared_with_user_id = $2
					  AND s.permission = 'editor'
				) THEN 'editor'
				WHEN EXISTS (
					SELECT 1
					FROM entity_shares s
					WHERE s.entity_type = 'playlist'
					  AND s.entity_id = p.id::text
					  AND (
						  s.shared_with_user_id = $2
						  OR ($3 <> '' AND s.share_token = $3)
					  )
				) THEN 'listener'
				ELSE ''
			END AS access_role
		FROM playlists p
		WHERE p.id = $1
		  AND (
			  p.owner_user_id = $2
			  OR EXISTS (
				  SELECT 1
				  FROM entity_shares s
				  WHERE s.entity_type = 'playlist'
				    AND s.entity_id = p.id::text
				    AND (
					    s.shared_with_user_id = $2
					    OR ($3 <> '' AND s.share_token = $3)
				    )
			  )
		  )
	`, id, userID, shareToken).Scan(
		&playlist.ID,
		&playlist.Name,
		&playlist.Description,
		&playlist.OwnerUserID,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
		&playlist.AccessRole,
	)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("get accessible playlist by id: %w", err)
	}
	return playlist, nil
}

func (r *PlaylistRepository) Create(ctx context.Context, playlist domain.Playlist) (domain.Playlist, error) {
	var created domain.Playlist
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO playlists (id, name, description, owner_user_id, created_at, updated_at)
		VALUES (
			COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()),
			$2,
			$3,
			$4,
			NOW(),
			NOW()
		)
		RETURNING id, name, COALESCE(description, ''), COALESCE(owner_user_id, ''), created_at, updated_at
	`, playlist.ID, playlist.Name, playlist.Description, playlist.OwnerUserID).Scan(
		&created.ID,
		&created.Name,
		&created.Description,
		&created.OwnerUserID,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("create playlist: %w", err)
	}
	created.AccessRole = "owner"
	return created, nil
}

func (r *PlaylistRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM playlists WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete playlist: %w", err)
	}
	return nil
}

func (r *PlaylistRepository) Rename(ctx context.Context, id, name string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE playlists
		SET name = $2, updated_at = NOW()
		WHERE id = $1
	`, id, name)
	if err != nil {
		return fmt.Errorf("rename playlist: %w", err)
	}
	return nil
}

func (r *PlaylistRepository) Update(ctx context.Context, id, name, description string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE playlists
		SET name = $2, description = $3, updated_at = NOW()
		WHERE id = $1
	`, id, name, description)
	if err != nil {
		return fmt.Errorf("update playlist: %w", err)
	}
	return nil
}

func (r *PlaylistRepository) CanEdit(ctx context.Context, id, userID string) (bool, error) {
	var canEdit bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM playlists p
			WHERE p.id = $1
			  AND (
				  p.owner_user_id = $2
				  OR EXISTS (
					  SELECT 1
					  FROM entity_shares s
					  WHERE s.entity_type = 'playlist'
					    AND s.entity_id = p.id::text
					    AND s.shared_with_user_id = $2
					    AND s.permission = 'editor'
				  )
			  )
		)
	`, id, userID).Scan(&canEdit)
	if err != nil {
		return false, fmt.Errorf("check playlist edit permission: %w", err)
	}
	return canEdit, nil
}

func (r *PlaylistRepository) IsOwner(ctx context.Context, id, userID string) (bool, error) {
	var isOwner bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM playlists WHERE id = $1 AND owner_user_id = $2
		)
	`, id, userID).Scan(&isOwner)
	if err != nil {
		return false, fmt.Errorf("check playlist owner: %w", err)
	}
	return isOwner, nil
}

func (r *PlaylistRepository) AddTrack(ctx context.Context, item domain.PlaylistTrack) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO playlist_tracks (playlist_id, track_id, position)
		VALUES ($1, $2, $3)
		ON CONFLICT (playlist_id, track_id) DO UPDATE
		SET position = EXCLUDED.position
	`, item.PlaylistID, item.TrackID, item.Position)
	if err != nil {
		return fmt.Errorf("add track to playlist: %w", err)
	}
	return nil
}

func (r *PlaylistRepository) RemoveTrack(ctx context.Context, playlistID string, trackID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM playlist_tracks
		WHERE playlist_id = $1 AND track_id = $2
	`, playlistID, trackID)
	if err != nil {
		return fmt.Errorf("remove track from playlist: %w", err)
	}
	return nil
}

func (r *PlaylistRepository) ListTracks(ctx context.Context, playlistID string) ([]domain.Track, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			t.id, t.title, t.album_id, t.artist_id, t.track_number, t.duration_seconds,
			t.file_path, t.genre, t.codec, t.bitrate, t.replay_gain_track, t.replay_gain_album, t.created_at, t.updated_at
		FROM playlist_tracks pt
		INNER JOIN tracks t ON t.id = pt.track_id
		WHERE pt.playlist_id = $1
		ORDER BY pt.position ASC
	`, playlistID)
	if err != nil {
		return nil, fmt.Errorf("list playlist tracks: %w", err)
	}
	defer rows.Close()

	tracks := make([]domain.Track, 0)
	for rows.Next() {
		var track domain.Track
		var durationSeconds int64
		if err := rows.Scan(
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
			&track.ReplayGainTrack,
			&track.ReplayGainAlbum,
			&track.CreatedAt,
			&track.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan playlist track: %w", err)
		}
		track.Duration = time.Duration(durationSeconds) * time.Second
		tracks = append(tracks, track)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playlist tracks rows: %w", err)
	}

	return tracks, nil
}
