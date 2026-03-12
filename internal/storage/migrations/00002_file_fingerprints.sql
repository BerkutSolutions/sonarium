-- +goose Up
CREATE TABLE library_file_fingerprints (
    file_path TEXT PRIMARY KEY,
    file_size BIGINT NOT NULL CHECK (file_size >= 0),
    mod_time TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_library_file_fingerprints_mod_time ON library_file_fingerprints(mod_time);

-- +goose Down
DROP INDEX IF EXISTS idx_library_file_fingerprints_mod_time;
DROP TABLE IF EXISTS library_file_fingerprints;
