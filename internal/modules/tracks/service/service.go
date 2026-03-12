package service

import (
	"context"
	"sort"
	"strings"

	"music-server/internal/domain"
)

type ListParams struct {
	Limit  int
	Offset int
	SortBy string
}

type Service struct {
	repository domain.TrackRepository
}

func New(repository domain.TrackRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, params ListParams) ([]domain.Track, error) {
	tracks, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	sortTracks(tracks, params.SortBy)
	return paginate(tracks, params.Offset, params.Limit), nil
}

func (s *Service) GetByID(ctx context.Context, id string) (domain.Track, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) ListByAlbumID(ctx context.Context, albumID string, params ListParams) ([]domain.Track, error) {
	tracks, err := s.repository.ListByAlbumID(ctx, albumID)
	if err != nil {
		return nil, err
	}
	sortTracks(tracks, params.SortBy)
	return paginate(tracks, params.Offset, params.Limit), nil
}

func (s *Service) Search(ctx context.Context, query string, params ListParams) ([]domain.Track, error) {
	tracks, err := s.repository.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	sortTracks(tracks, params.SortBy)
	return paginate(tracks, params.Offset, params.Limit), nil
}

func sortTracks(tracks []domain.Track, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "created_at":
		sort.Slice(tracks, func(i, j int) bool {
			return tracks[i].CreatedAt.Before(tracks[j].CreatedAt)
		})
	default:
		sort.Slice(tracks, func(i, j int) bool {
			return strings.ToLower(tracks[i].Title) < strings.ToLower(tracks[j].Title)
		})
	}
}

func paginate(items []domain.Track, offset, limit int) []domain.Track {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []domain.Track{}
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}
