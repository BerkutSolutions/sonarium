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
	repository domain.AlbumRepository
}

func New(repository domain.AlbumRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, params ListParams) ([]domain.Album, error) {
	albums, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	sortAlbums(albums, params.SortBy)
	return paginate(albums, params.Offset, params.Limit), nil
}

func (s *Service) GetByID(ctx context.Context, id string) (domain.Album, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) ListByArtistID(ctx context.Context, artistID string, params ListParams) ([]domain.Album, error) {
	albums, err := s.repository.ListByArtistID(ctx, artistID)
	if err != nil {
		return nil, err
	}
	sortAlbums(albums, params.SortBy)
	return paginate(albums, params.Offset, params.Limit), nil
}

func (s *Service) Search(ctx context.Context, query string, params ListParams) ([]domain.Album, error) {
	albums, err := s.repository.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	sortAlbums(albums, params.SortBy)
	return paginate(albums, params.Offset, params.Limit), nil
}

func sortAlbums(albums []domain.Album, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "year":
		sort.Slice(albums, func(i, j int) bool {
			return albums[i].Year < albums[j].Year
		})
	case "created_at":
		sort.Slice(albums, func(i, j int) bool {
			return albums[i].CreatedAt.Before(albums[j].CreatedAt)
		})
	default:
		sort.Slice(albums, func(i, j int) bool {
			return strings.ToLower(albums[i].Title) < strings.ToLower(albums[j].Title)
		})
	}
}

func paginate(items []domain.Album, offset, limit int) []domain.Album {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []domain.Album{}
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}
