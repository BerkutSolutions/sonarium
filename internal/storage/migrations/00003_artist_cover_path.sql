-- +goose Up
ALTER TABLE artists
ADD COLUMN IF NOT EXISTS cover_path TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE artists
DROP COLUMN IF EXISTS cover_path;
