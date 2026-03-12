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
	repository domain.ArtistRepository
}

func New(repository domain.ArtistRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, params ListParams) ([]domain.Artist, error) {
	artists, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(params.SortBy) {
	case "created_at":
		sort.Slice(artists, func(i, j int) bool {
			return artists[i].CreatedAt.Before(artists[j].CreatedAt)
		})
	default:
		sort.Slice(artists, func(i, j int) bool {
			return strings.ToLower(artists[i].Name) < strings.ToLower(artists[j].Name)
		})
	}

	return paginateArtists(artists, params.Offset, params.Limit), nil
}

func (s *Service) GetByID(ctx context.Context, id string) (domain.Artist, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) Search(ctx context.Context, query string, params ListParams) ([]domain.Artist, error) {
	artists, err := s.repository.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(params.SortBy) {
	case "created_at":
		sort.Slice(artists, func(i, j int) bool {
			return artists[i].CreatedAt.Before(artists[j].CreatedAt)
		})
	default:
		sort.Slice(artists, func(i, j int) bool {
			return strings.ToLower(artists[i].Name) < strings.ToLower(artists[j].Name)
		})
	}

	return paginateArtists(artists, params.Offset, params.Limit), nil
}

func paginateArtists(items []domain.Artist, offset, limit int) []domain.Artist {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []domain.Artist{}
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}
