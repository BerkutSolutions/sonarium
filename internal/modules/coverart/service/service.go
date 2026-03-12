package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"music-server/internal/domain"
	covercache "music-server/internal/modules/coverart/cache"
	coverextractor "music-server/internal/modules/coverart/extractor"
)

var (
	ErrAlbumNotFound  = errors.New("album not found")
	ErrArtistNotFound = errors.New("artist not found")
)

type Service struct {
	artistRepository domain.ArtistRepository
	albumRepository  domain.AlbumRepository
	cache            *covercache.Cache
	extractor        *coverextractor.Extractor
}

func New(
	artistRepository domain.ArtistRepository,
	albumRepository domain.AlbumRepository,
	cache *covercache.Cache,
	extractor *coverextractor.Extractor,
) *Service {
	return &Service{
		artistRepository: artistRepository,
		albumRepository:  albumRepository,
		cache:            cache,
		extractor:        extractor,
	}
}

func (s *Service) ResolveAndAttach(
	ctx context.Context,
	artist domain.Artist,
	album domain.Album,
	trackPath string,
	embeddedData []byte,
	embeddedMIME string,
) (domain.Artist, domain.Album, error) {
	result, err := s.extractor.Extract(trackPath, embeddedData, embeddedMIME)
	if err != nil {
		return artist, album, err
	}
	if result == nil {
		return artist, album, nil
	}

	originalPath, err := s.cache.SaveOriginal(result.Data, result.MIME)
	if err != nil {
		return artist, album, err
	}

	if strings.TrimSpace(album.CoverPath) == "" {
		album.CoverPath = originalPath
		if err := s.albumRepository.Upsert(ctx, album); err != nil {
			return artist, album, fmt.Errorf("upsert album cover path: %w", err)
		}
	}

	if strings.TrimSpace(artist.CoverPath) == "" {
		artist.CoverPath = originalPath
		if err := s.artistRepository.Upsert(ctx, artist); err != nil {
			return artist, album, fmt.Errorf("upsert artist cover path: %w", err)
		}
	}

	return artist, album, nil
}

func (s *Service) AlbumOriginal(ctx context.Context, albumID string, withPlaceholder bool) (string, string, error) {
	album, err := s.albumRepository.GetByID(ctx, albumID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if withPlaceholder {
				placeholder, placeholderErr := s.cache.PlaceholderPath(0)
				return placeholder, "image/png", placeholderErr
			}
			return "", "", ErrAlbumNotFound
		}
		return "", "", err
	}
	if strings.TrimSpace(album.CoverPath) != "" {
		if _, statErr := os.Stat(album.CoverPath); statErr == nil {
			return album.CoverPath, mimeByPath(album.CoverPath), nil
		}
	}
	if withPlaceholder {
		placeholder, err := s.cache.PlaceholderPath(0)
		return placeholder, "image/png", err
	}
	return "", "", os.ErrNotExist
}

func (s *Service) AlbumThumb(ctx context.Context, albumID string, size int, withPlaceholder bool) (string, string, error) {
	if strings.TrimSpace(albumID) == "" && withPlaceholder {
		placeholder, err := s.cache.PlaceholderPath(size)
		return placeholder, "image/jpeg", err
	}
	original, _, err := s.AlbumOriginal(ctx, albumID, withPlaceholder)
	if err != nil {
		if withPlaceholder && errors.Is(err, ErrAlbumNotFound) {
			placeholder, placeholderErr := s.cache.PlaceholderPath(size)
			return placeholder, "image/jpeg", placeholderErr
		}
		return "", "", err
	}
	thumb, err := s.cache.ThumbPath(original, size)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unsupported thumb size") {
			return "", "", err
		}
		return "", "", err
	}
	return thumb, "image/jpeg", nil
}

func (s *Service) ArtistOriginal(ctx context.Context, artistID string, withPlaceholder bool) (string, string, error) {
	artist, err := s.artistRepository.GetByID(ctx, artistID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrArtistNotFound
		}
		return "", "", err
	}
	if strings.TrimSpace(artist.CoverPath) != "" {
		if _, statErr := os.Stat(artist.CoverPath); statErr == nil {
			return artist.CoverPath, mimeByPath(artist.CoverPath), nil
		}
	}
	if withPlaceholder {
		placeholder, err := s.cache.PlaceholderPath(0)
		return placeholder, "image/png", err
	}
	return "", "", os.ErrNotExist
}

func (s *Service) SaveArtistCover(ctx context.Context, artistID string, data []byte, mimeType string) error {
	artist, err := s.artistRepository.GetByID(ctx, artistID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrArtistNotFound
		}
		return err
	}
	originalPath, err := s.cache.SaveOriginal(data, mimeType)
	if err != nil {
		return err
	}
	artist.CoverPath = originalPath
	return s.artistRepository.Upsert(ctx, artist)
}

func mimeByPath(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	default:
		return "image/jpeg"
	}
}
