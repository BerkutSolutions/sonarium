-- +goose Up
ALTER TABLE tracks
    ADD COLUMN IF NOT EXISTS genre TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_tracks_genre ON tracks(genre);

-- +goose Down
DROP INDEX IF EXISTS idx_tracks_genre;

ALTER TABLE tracks
    DROP COLUMN IF EXISTS genre;
