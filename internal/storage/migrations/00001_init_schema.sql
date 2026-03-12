-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE artists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    artist_id UUID NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    year INT NOT NULL DEFAULT 0 CHECK (year >= 0 AND year <= 3000),
    cover_path TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tracks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    album_id UUID NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    artist_id UUID NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    track_number INT NOT NULL CHECK (track_number > 0),
    duration_seconds BIGINT NOT NULL CHECK (duration_seconds > 0),
    file_path TEXT NOT NULL,
    codec TEXT NOT NULL DEFAULT '',
    bitrate INT NOT NULL CHECK (bitrate > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE playlists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE playlist_tracks (
    playlist_id UUID NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    position INT NOT NULL CHECK (position > 0),
    PRIMARY KEY (playlist_id, track_id)
);

CREATE TABLE library (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id = TRUE),
    root_path TEXT NOT NULL,
    last_scan_at TIMESTAMPTZ
);

INSERT INTO library (id, root_path, last_scan_at)
VALUES (TRUE, '/music', NULL)
ON CONFLICT (id) DO NOTHING;

CREATE INDEX idx_artists_name ON artists(name);
CREATE INDEX idx_albums_artist_id ON albums(artist_id);
CREATE INDEX idx_tracks_album_id ON tracks(album_id);
CREATE INDEX idx_playlist_tracks_position ON playlist_tracks(position);
CREATE UNIQUE INDEX idx_playlist_tracks_playlist_position ON playlist_tracks(playlist_id, position);

-- +goose Down
DROP INDEX IF EXISTS idx_playlist_tracks_playlist_position;
DROP INDEX IF EXISTS idx_playlist_tracks_position;
DROP INDEX IF EXISTS idx_tracks_album_id;
DROP INDEX IF EXISTS idx_albums_artist_id;
DROP INDEX IF EXISTS idx_artists_name;

DROP TABLE IF EXISTS library;
DROP TABLE IF EXISTS playlist_tracks;
DROP TABLE IF EXISTS playlists;
DROP TABLE IF EXISTS tracks;
DROP TABLE IF EXISTS albums;
DROP TABLE IF EXISTS artists;
