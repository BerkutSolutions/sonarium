-- +goose Up
ALTER TABLE tracks
    ADD COLUMN IF NOT EXISTS file_size_bytes BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS uploaded_by_user_id TEXT REFERENCES app_users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_tracks_uploaded_by_user_id ON tracks(uploaded_by_user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_tracks_uploaded_by_user_id;
ALTER TABLE tracks
    DROP COLUMN IF EXISTS uploaded_by_user_id,
    DROP COLUMN IF EXISTS file_size_bytes;
