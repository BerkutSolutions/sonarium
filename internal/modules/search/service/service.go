package service

import (
	"context"
	"sort"
	"strings"

	"music-server/internal/domain"
)

type Params struct {
	Query  string
	Limit  int
	Offset int
	SortBy string
}

type Result struct {
	Artists []domain.Artist `json:"artists"`
	Albums  []domain.Album  `json:"albums"`
	Tracks  []domain.Track  `json:"tracks"`
}

type Service struct {
	artistRepository domain.ArtistRepository
	albumRepository  domain.AlbumRepository
	trackRepository  domain.TrackRepository
}

func New(artistRepository domain.ArtistRepository, albumRepository domain.AlbumRepository, trackRepository domain.TrackRepository) *Service {
	return &Service{
		artistRepository: artistRepository,
		albumRepository:  albumRepository,
		trackRepository:  trackRepository,
	}
}

func (s *Service) Search(ctx context.Context, params Params) (Result, error) {
	artists, err := s.artistRepository.Search(ctx, params.Query)
	if err != nil {
		return Result{}, err
	}
	albums, err := s.albumRepository.Search(ctx, params.Query)
	if err != nil {
		return Result{}, err
	}
	tracks, err := s.trackRepository.Search(ctx, params.Query)
	if err != nil {
		return Result{}, err
	}

	sortArtists(artists, params.SortBy)
	sortAlbums(albums, params.SortBy)
	sortTracks(tracks, params.SortBy)

	return Result{
		Artists: paginateArtists(artists, params.Offset, params.Limit),
		Albums:  paginateAlbums(albums, params.Offset, params.Limit),
		Tracks:  paginateTracks(tracks, params.Offset, params.Limit),
	}, nil
}

func sortArtists(items []domain.Artist, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "created_at":
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	default:
		sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name) })
	}
}

func sortAlbums(items []domain.Album, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "year":
		sort.Slice(items, func(i, j int) bool { return items[i].Year < items[j].Year })
	case "created_at":
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	default:
		sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title) })
	}
}

func sortTracks(items []domain.Track, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "created_at":
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	default:
		sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title) })
	}
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

func paginateAlbums(items []domain.Album, offset, limit int) []domain.Album {
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

func paginateTracks(items []domain.Track, offset, limit int) []domain.Track {
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
