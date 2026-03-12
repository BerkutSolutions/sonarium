-- +goose Up
ALTER TABLE app_users
    ADD COLUMN IF NOT EXISTS profile_public BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE playlists
    ADD COLUMN IF NOT EXISTS owner_user_id TEXT REFERENCES app_users(id) ON DELETE SET NULL;

UPDATE playlists
SET owner_user_id = COALESCE(
    owner_user_id,
    (SELECT id FROM app_users ORDER BY created_at ASC LIMIT 1)
)
WHERE owner_user_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_playlists_owner_user_id ON playlists(owner_user_id);

CREATE TABLE IF NOT EXISTS entity_shares (
    id TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    owner_user_id TEXT NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    shared_with_user_id TEXT REFERENCES app_users(id) ON DELETE CASCADE,
    permission TEXT NOT NULL DEFAULT 'viewer',
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    share_token TEXT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_entity_shares_owner_user_id ON entity_shares(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_entity_shares_shared_with_user_id ON entity_shares(shared_with_user_id);
CREATE INDEX IF NOT EXISTS idx_entity_shares_entity ON entity_shares(entity_type, entity_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_entity_shares_unique_user
    ON entity_shares(entity_type, entity_id, owner_user_id, shared_with_user_id)
    WHERE shared_with_user_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_entity_shares_unique_public
    ON entity_shares(entity_type, entity_id, owner_user_id, is_public)
    WHERE is_public = TRUE;

-- +goose Down
DROP INDEX IF EXISTS idx_entity_shares_unique_public;
DROP INDEX IF EXISTS idx_entity_shares_unique_user;
DROP INDEX IF EXISTS idx_entity_shares_entity;
DROP INDEX IF EXISTS idx_entity_shares_shared_with_user_id;
DROP INDEX IF EXISTS idx_entity_shares_owner_user_id;
DROP TABLE IF EXISTS entity_shares;

DROP INDEX IF EXISTS idx_playlists_owner_user_id;
ALTER TABLE playlists DROP COLUMN IF EXISTS owner_user_id;
ALTER TABLE app_users DROP COLUMN IF EXISTS profile_public;
