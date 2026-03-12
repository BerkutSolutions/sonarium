-- +goose Up
ALTER TABLE tracks
    ADD COLUMN IF NOT EXISTS replay_gain_track DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS replay_gain_album DOUBLE PRECISION NOT NULL DEFAULT 0;

ALTER TABLE albums
    ADD COLUMN IF NOT EXISTS random_key DOUBLE PRECISION NOT NULL DEFAULT random();

CREATE INDEX IF NOT EXISTS idx_albums_random_key ON albums(random_key);

ALTER TABLE library_file_fingerprints
    ADD COLUMN IF NOT EXISTS fingerprint_hash TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_library_file_fingerprints_hash ON library_file_fingerprints(fingerprint_hash);

CREATE TABLE IF NOT EXISTS user_favorite_tracks (
    user_id TEXT NOT NULL,
    track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, track_id)
);

CREATE TABLE IF NOT EXISTS user_favorite_albums (
    user_id TEXT NOT NULL,
    album_id UUID NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, album_id)
);

CREATE TABLE IF NOT EXISTS user_favorite_artists (
    user_id TEXT NOT NULL,
    artist_id UUID NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, artist_id)
);

CREATE TABLE IF NOT EXISTS play_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    played_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    position_seconds INT NOT NULL DEFAULT 0 CHECK (position_seconds >= 0),
    context_type TEXT NOT NULL DEFAULT '',
    context_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_play_history_user_played_at ON play_history(user_id, played_at DESC);
CREATE INDEX IF NOT EXISTS idx_play_history_user_track_played_at ON play_history(user_id, track_id, played_at DESC);

CREATE TABLE IF NOT EXISTS library_album_stats (
    album_id UUID PRIMARY KEY REFERENCES albums(id) ON DELETE CASCADE,
    tracks_count INT NOT NULL DEFAULT 0,
    total_duration_seconds BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS library_artist_stats (
    artist_id UUID PRIMARY KEY REFERENCES artists(id) ON DELETE CASCADE,
    albums_count INT NOT NULL DEFAULT 0,
    tracks_count INT NOT NULL DEFAULT 0,
    total_duration_seconds BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_favorite_tracks_user ON user_favorite_tracks(user_id);
CREATE INDEX IF NOT EXISTS idx_user_favorite_albums_user ON user_favorite_albums(user_id);
CREATE INDEX IF NOT EXISTS idx_user_favorite_artists_user ON user_favorite_artists(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_user_favorite_artists_user;
DROP INDEX IF EXISTS idx_user_favorite_albums_user;
DROP INDEX IF EXISTS idx_user_favorite_tracks_user;
DROP TABLE IF EXISTS library_artist_stats;
DROP TABLE IF EXISTS library_album_stats;
DROP INDEX IF EXISTS idx_play_history_user_track_played_at;
DROP INDEX IF EXISTS idx_play_history_user_played_at;
DROP TABLE IF EXISTS play_history;
DROP TABLE IF EXISTS user_favorite_artists;
DROP TABLE IF EXISTS user_favorite_albums;
DROP TABLE IF EXISTS user_favorite_tracks;
DROP INDEX IF EXISTS idx_library_file_fingerprints_hash;
ALTER TABLE library_file_fingerprints DROP COLUMN IF EXISTS fingerprint_hash;
DROP INDEX IF EXISTS idx_albums_random_key;
ALTER TABLE albums DROP COLUMN IF EXISTS random_key;
ALTER TABLE tracks
    DROP COLUMN IF EXISTS replay_gain_track,
    DROP COLUMN IF EXISTS replay_gain_album;
