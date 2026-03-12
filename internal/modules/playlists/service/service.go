package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"music-server/internal/domain"
	authservice "music-server/internal/modules/auth/service"
)

type ListParams struct {
	Limit  int
	Offset int
	SortBy string
}

type Service struct {
	repository domain.PlaylistRepository
}

func New(repository domain.PlaylistRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, current *authservice.User, params ListParams) ([]domain.Playlist, error) {
	var (
		playlists []domain.Playlist
		err       error
	)
	if current == nil {
		playlists, err = s.repository.List(ctx)
	} else {
		playlists, err = s.repository.ListAccessible(ctx, current.ID)
	}
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(params.SortBy) {
	case "created_at":
		sort.Slice(playlists, func(i, j int) bool {
			return playlists[i].CreatedAt.Before(playlists[j].CreatedAt)
		})
	default:
		sort.Slice(playlists, func(i, j int) bool {
			return strings.ToLower(playlists[i].Name) < strings.ToLower(playlists[j].Name)
		})
	}

	return paginate(playlists, params.Offset, params.Limit), nil
}

func (s *Service) Create(ctx context.Context, current *authservice.User, name string) (domain.Playlist, error) {
	if current == nil {
		return domain.Playlist{}, authservice.ErrUnauthorized
	}
	playlist := domain.Playlist{
		Name:        name,
		Description: "",
		OwnerUserID: current.ID,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	return s.repository.Create(ctx, playlist)
}

func (s *Service) GetByID(ctx context.Context, current *authservice.User, id, shareToken string) (domain.Playlist, error) {
	if current == nil {
		return s.repository.GetByID(ctx, id)
	}
	return s.repository.GetAccessibleByID(ctx, id, current.ID, shareToken)
}

func (s *Service) ListTracks(ctx context.Context, current *authservice.User, playlistID, shareToken string) ([]domain.Track, error) {
	if current != nil {
		if _, err := s.GetByID(ctx, current, playlistID, shareToken); err != nil {
			return nil, err
		}
	}
	return s.repository.ListTracks(ctx, playlistID)
}

func (s *Service) AddTrack(ctx context.Context, current *authservice.User, playlistID string, trackID string, position int) error {
	if current == nil {
		return authservice.ErrUnauthorized
	}
	if current.Role == authservice.RoleAdmin {
		return s.repository.AddTrack(ctx, domain.PlaylistTrack{
			PlaylistID: playlistID,
			TrackID:    trackID,
			Position:   position,
		})
	}
	canEdit, err := s.repository.CanEdit(ctx, playlistID, current.ID)
	if err != nil {
		return err
	}
	if !canEdit {
		return authservice.ErrForbidden
	}
	return s.repository.AddTrack(ctx, domain.PlaylistTrack{
		PlaylistID: playlistID,
		TrackID:    trackID,
		Position:   position,
	})
}

func (s *Service) RemoveTrack(ctx context.Context, current *authservice.User, playlistID string, trackID string) error {
	if current == nil {
		return authservice.ErrUnauthorized
	}
	if current.Role == authservice.RoleAdmin {
		return s.repository.RemoveTrack(ctx, playlistID, trackID)
	}
	canEdit, err := s.repository.CanEdit(ctx, playlistID, current.ID)
	if err != nil {
		return err
	}
	if !canEdit {
		return authservice.ErrForbidden
	}
	return s.repository.RemoveTrack(ctx, playlistID, trackID)
}

func (s *Service) Delete(ctx context.Context, current *authservice.User, id string) error {
	if current == nil {
		return authservice.ErrUnauthorized
	}
	isOwner, err := s.repository.IsOwner(ctx, id, current.ID)
	if err != nil {
		return err
	}
	if !isOwner && current.Role != authservice.RoleAdmin {
		return authservice.ErrForbidden
	}
	return s.repository.Delete(ctx, id)
}

func (s *Service) Rename(ctx context.Context, current *authservice.User, id, name string) error {
	if current == nil {
		return authservice.ErrUnauthorized
	}
	if current.Role == authservice.RoleAdmin {
		return s.repository.Rename(ctx, id, name)
	}
	canEdit, err := s.repository.CanEdit(ctx, id, current.ID)
	if err != nil {
		return err
	}
	if !canEdit {
		return authservice.ErrForbidden
	}
	return s.repository.Rename(ctx, id, name)
}

func (s *Service) Update(ctx context.Context, current *authservice.User, id, name, description string) error {
	if current == nil {
		return authservice.ErrUnauthorized
	}
	if current.Role == authservice.RoleAdmin {
		return s.repository.Update(ctx, id, name, description)
	}
	canEdit, err := s.repository.CanEdit(ctx, id, current.ID)
	if err != nil {
		return err
	}
	if !canEdit {
		return authservice.ErrForbidden
	}
	return s.repository.Update(ctx, id, name, description)
}

func paginate(items []domain.Playlist, offset, limit int) []domain.Playlist {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []domain.Playlist{}
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}
