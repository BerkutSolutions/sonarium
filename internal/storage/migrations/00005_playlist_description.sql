-- +goose Up
ALTER TABLE playlists
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE playlists
    DROP COLUMN IF EXISTS description;
