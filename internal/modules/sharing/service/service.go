package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	authservice "music-server/internal/modules/auth/service"
)

var ErrInvalidPermission = errors.New("invalid permission")

type Share struct {
	ID                   string    `json:"id"`
	EntityType           string    `json:"entity_type"`
	EntityID             string    `json:"entity_id"`
	OwnerUserID          string    `json:"owner_user_id"`
	SharedWithUserID     string    `json:"shared_with_user_id"`
	Permission           string    `json:"permission"`
	IsPublic             bool      `json:"is_public"`
	ShareToken           string    `json:"share_token"`
	RecipientUsername    string    `json:"recipient_username"`
	RecipientDisplayName string    `json:"recipient_display_name"`
	CreatedAt            time.Time `json:"created_at"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListEntityShares(ctx context.Context, current *authservice.User, entityType, entityID string) ([]Share, error) {
	if err := s.ensureCanManageEntity(ctx, current, entityType, entityID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			s.id,
			s.entity_type,
			s.entity_id,
			s.owner_user_id,
			COALESCE(s.shared_with_user_id, ''),
			s.permission,
			s.is_public,
			COALESCE(s.share_token, ''),
			COALESCE(u.username, ''),
			COALESCE(u.display_name, ''),
			s.created_at
		FROM entity_shares s
		LEFT JOIN app_users u ON u.id = s.shared_with_user_id
		WHERE s.entity_type = $1
		  AND s.entity_id = $2
		  AND (
			  $3 = 'admin'
			  OR s.owner_user_id = $4
			  OR ($1 = 'playlist' AND EXISTS (
				  SELECT 1 FROM playlists p WHERE p.id::text = $2 AND p.owner_user_id = $4
			  ))
		  )
		ORDER BY s.is_public DESC, u.display_name ASC, u.username ASC, s.created_at ASC
	`, entityType, entityID, current.Role, current.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shares := make([]Share, 0)
	for rows.Next() {
		var share Share
		if err := rows.Scan(
			&share.ID,
			&share.EntityType,
			&share.EntityID,
			&share.OwnerUserID,
			&share.SharedWithUserID,
			&share.Permission,
			&share.IsPublic,
			&share.ShareToken,
			&share.RecipientUsername,
			&share.RecipientDisplayName,
			&share.CreatedAt,
		); err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}
	return shares, rows.Err()
}

func (s *Service) UpsertUserShare(ctx context.Context, current *authservice.User, entityType, entityID, sharedWithUserID, permission string) (Share, error) {
	if err := s.ensureCanManageEntity(ctx, current, entityType, entityID); err != nil {
		return Share{}, err
	}
	if strings.TrimSpace(sharedWithUserID) == "" {
		return Share{}, authservice.ErrForbidden
	}
	permission = normalizePermission(entityType, permission)
	if permission == "" {
		return Share{}, ErrInvalidPermission
	}
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM entity_shares
		WHERE entity_type = $1
		  AND entity_id = $2
		  AND owner_user_id = $3
		  AND shared_with_user_id = $4
	`, entityType, entityID, current.ID, sharedWithUserID); err != nil {
		return Share{}, err
	}
	shareID := mustRandomID(16)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO entity_shares (
			id,
			entity_type,
			entity_id,
			owner_user_id,
			shared_with_user_id,
			permission,
			is_public,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, FALSE, NOW())
	`, shareID, entityType, entityID, current.ID, sharedWithUserID, permission)
	if err != nil {
		return Share{}, err
	}
	return s.lookupUserShare(ctx, current.ID, entityType, entityID, sharedWithUserID)
}

func (s *Service) SetPublicShare(ctx context.Context, current *authservice.User, entityType, entityID string, enabled bool) (*Share, error) {
	if err := s.ensureCanManageEntity(ctx, current, entityType, entityID); err != nil {
		return nil, err
	}
	if !enabled {
		if _, err := s.db.ExecContext(ctx, `
			DELETE FROM entity_shares
			WHERE entity_type = $1
			  AND entity_id = $2
			  AND owner_user_id = $3
			  AND is_public = TRUE
		`, entityType, entityID, current.ID); err != nil {
			return nil, err
		}
		return nil, nil
	}

	token := mustRandomID(24)
	shareID := mustRandomID(16)
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM entity_shares
		WHERE entity_type = $1
		  AND entity_id = $2
		  AND owner_user_id = $3
		  AND is_public = TRUE
	`, entityType, entityID, current.ID); err != nil {
		return nil, err
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO entity_shares (
			id,
			entity_type,
			entity_id,
			owner_user_id,
			shared_with_user_id,
			permission,
			is_public,
			share_token,
			updated_at
		)
		VALUES ($1, $2, $3, $4, NULL, 'viewer', TRUE, $5, NOW())
	`, shareID, entityType, entityID, current.ID, token)
	if err != nil {
		return nil, err
	}
	share, err := s.lookupPublicShare(ctx, current.ID, entityType, entityID)
	if err != nil {
		return nil, err
	}
	return &share, nil
}

func (s *Service) DeleteShare(ctx context.Context, current *authservice.User, shareID string) error {
	if current == nil {
		return authservice.ErrUnauthorized
	}
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM entity_shares
		WHERE id = $1
		  AND ($2 = 'admin' OR owner_user_id = $3)
	`, shareID, current.Role, current.ID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return authservice.ErrForbidden
	}
	return nil
}

func (s *Service) ListReceivedShares(ctx context.Context, current *authservice.User, userID string) ([]Share, error) {
	if current == nil {
		return nil, authservice.ErrUnauthorized
	}
	if current.ID != userID && current.Role != authservice.RoleAdmin {
		return nil, authservice.ErrForbidden
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			s.id,
			s.entity_type,
			s.entity_id,
			s.owner_user_id,
			COALESCE(s.shared_with_user_id, ''),
			s.permission,
			s.is_public,
			COALESCE(s.share_token, ''),
			COALESCE(u.username, ''),
			COALESCE(u.display_name, ''),
			s.created_at
		FROM entity_shares s
		LEFT JOIN app_users u ON u.id = s.owner_user_id
		WHERE s.shared_with_user_id = $1
		ORDER BY s.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shares := make([]Share, 0)
	for rows.Next() {
		var share Share
		if err := rows.Scan(
			&share.ID,
			&share.EntityType,
			&share.EntityID,
			&share.OwnerUserID,
			&share.SharedWithUserID,
			&share.Permission,
			&share.IsPublic,
			&share.ShareToken,
			&share.RecipientUsername,
			&share.RecipientDisplayName,
			&share.CreatedAt,
		); err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}
	return shares, rows.Err()
}

func (s *Service) ensureCanManageEntity(ctx context.Context, current *authservice.User, entityType, entityID string) error {
	if current == nil {
		return authservice.ErrUnauthorized
	}
	entityType = strings.TrimSpace(strings.ToLower(entityType))
	entityID = strings.TrimSpace(entityID)
	if entityType == "" || entityID == "" {
		return authservice.ErrForbidden
	}
	if current.Role == authservice.RoleAdmin {
		return nil
	}
	if entityType != "playlist" {
		return nil
	}
	var ownerUserID string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(owner_user_id, '') FROM playlists WHERE id = $1`, entityID).Scan(&ownerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return authservice.ErrForbidden
		}
		return err
	}
	if ownerUserID != current.ID {
		return authservice.ErrForbidden
	}
	return nil
}

func (s *Service) lookupUserShare(ctx context.Context, ownerUserID, entityType, entityID, sharedWithUserID string) (Share, error) {
	var share Share
	err := s.db.QueryRowContext(ctx, `
		SELECT
			s.id,
			s.entity_type,
			s.entity_id,
			s.owner_user_id,
			COALESCE(s.shared_with_user_id, ''),
			s.permission,
			s.is_public,
			COALESCE(s.share_token, ''),
			COALESCE(u.username, ''),
			COALESCE(u.display_name, ''),
			s.created_at
		FROM entity_shares s
		LEFT JOIN app_users u ON u.id = s.shared_with_user_id
		WHERE s.entity_type = $1
		  AND s.entity_id = $2
		  AND s.owner_user_id = $3
		  AND s.shared_with_user_id = $4
	`, entityType, entityID, ownerUserID, sharedWithUserID).Scan(
		&share.ID,
		&share.EntityType,
		&share.EntityID,
		&share.OwnerUserID,
		&share.SharedWithUserID,
		&share.Permission,
		&share.IsPublic,
		&share.ShareToken,
		&share.RecipientUsername,
		&share.RecipientDisplayName,
		&share.CreatedAt,
	)
	return share, err
}

func (s *Service) lookupPublicShare(ctx context.Context, ownerUserID, entityType, entityID string) (Share, error) {
	var share Share
	err := s.db.QueryRowContext(ctx, `
		SELECT id, entity_type, entity_id, owner_user_id, COALESCE(shared_with_user_id, ''), permission, is_public, COALESCE(share_token, ''), '', '', created_at
		FROM entity_shares
		WHERE entity_type = $1
		  AND entity_id = $2
		  AND owner_user_id = $3
		  AND is_public = TRUE
	`, entityType, entityID, ownerUserID).Scan(
		&share.ID,
		&share.EntityType,
		&share.EntityID,
		&share.OwnerUserID,
		&share.SharedWithUserID,
		&share.Permission,
		&share.IsPublic,
		&share.ShareToken,
		&share.RecipientUsername,
		&share.RecipientDisplayName,
		&share.CreatedAt,
	)
	return share, err
}

func normalizePermission(entityType, permission string) string {
	entityType = strings.TrimSpace(strings.ToLower(entityType))
	permission = strings.TrimSpace(strings.ToLower(permission))
	switch entityType {
	case "playlist":
		if permission == "editor" {
			return "editor"
		}
		return "listener"
	case "album", "artist", "track":
		return "viewer"
	default:
		return ""
	}
}

func mustRandomID(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
