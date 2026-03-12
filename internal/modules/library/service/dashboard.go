package service

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"time"

	"music-server/internal/domain"
	coverservice "music-server/internal/modules/coverart/service"
	libraryrepo "music-server/internal/modules/library/repository"
)

const DefaultUserID = "default"

type DashboardService struct {
	repository *libraryrepo.Repository
	ttl        time.Duration
	artistRepo domain.ArtistRepository
	trackRepo  domain.TrackRepository
	coverSvc   *coverservice.Service

	mu          sync.RWMutex
	cacheUserID string
	cacheUntil  time.Time
	cacheValue  libraryrepo.HomeData
}

func NewDashboardService(repository *libraryrepo.Repository, artistRepo domain.ArtistRepository, trackRepo domain.TrackRepository, coverSvc *coverservice.Service, ttl time.Duration) *DashboardService {
	if ttl <= 0 {
		ttl = 15 * time.Second
	}
	return &DashboardService{
		repository: repository,
		ttl:        ttl,
		artistRepo: artistRepo,
		trackRepo:  trackRepo,
		coverSvc:   coverSvc,
	}
}

func (s *DashboardService) Home(ctx context.Context, userID string, limit int) (libraryrepo.HomeData, error) {
	if userID == "" {
		userID = DefaultUserID
	}
	now := time.Now()
	s.mu.RLock()
	if s.cacheUserID == userID && now.Before(s.cacheUntil) {
		value := s.cacheValue
		s.mu.RUnlock()
		return value, nil
	}
	s.mu.RUnlock()

	home, err := s.repository.HomeData(ctx, userID, limit)
	if err != nil {
		return libraryrepo.HomeData{}, err
	}
	s.mu.Lock()
	s.cacheUserID = userID
	s.cacheUntil = now.Add(s.ttl)
	s.cacheValue = home
	s.mu.Unlock()
	return home, nil
}

func (s *DashboardService) Invalidate() {
	s.mu.Lock()
	s.cacheUntil = time.Time{}
	s.mu.Unlock()
}

func (s *DashboardService) RandomAlbums(ctx context.Context, limit int) ([]libraryrepo.AlbumCard, error) {
	return s.repository.RandomAlbums(ctx, limit)
}

func (s *DashboardService) ToggleFavoriteTrack(ctx context.Context, userID, trackID string) (bool, error) {
	if userID == "" {
		userID = DefaultUserID
	}
	value, err := s.repository.ToggleFavoriteTrack(ctx, userID, trackID)
	if err == nil {
		s.Invalidate()
	}
	return value, err
}

func (s *DashboardService) ToggleFavoriteAlbum(ctx context.Context, userID, albumID string) (bool, error) {
	if userID == "" {
		userID = DefaultUserID
	}
	value, err := s.repository.ToggleFavoriteAlbum(ctx, userID, albumID)
	if err == nil {
		s.Invalidate()
	}
	return value, err
}

func (s *DashboardService) ToggleFavoriteArtist(ctx context.Context, userID, artistID string) (bool, error) {
	if userID == "" {
		userID = DefaultUserID
	}
	value, err := s.repository.ToggleFavoriteArtist(ctx, userID, artistID)
	if err == nil {
		s.Invalidate()
	}
	return value, err
}

func (s *DashboardService) RecordPlayEvent(ctx context.Context, userID string, event libraryrepo.PlayEvent) error {
	if userID == "" {
		userID = DefaultUserID
	}
	if err := s.repository.RecordPlayEvent(ctx, userID, event); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

func (s *DashboardService) ArtistAlbumCounts(ctx context.Context) ([]libraryrepo.ArtistAlbumCount, error) {
	return s.repository.ArtistAlbumCounts(ctx)
}

func (s *DashboardService) DeleteTrack(ctx context.Context, trackID string) error {
	track, err := s.trackRepo.GetByID(ctx, trackID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err := s.repository.DeleteTrack(ctx, trackID); err != nil {
		return err
	}
	if track.FilePath != "" {
		if err := os.Remove(track.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	s.Invalidate()
	return nil
}

func (s *DashboardService) RenameTrack(ctx context.Context, trackID, title string) error {
	if err := s.repository.RenameTrack(ctx, trackID, title); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

func (s *DashboardService) DeleteAlbum(ctx context.Context, albumID string) error {
	if err := s.repository.DeleteAlbum(ctx, albumID); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

func (s *DashboardService) RenameAlbum(ctx context.Context, albumID, title string) error {
	if err := s.repository.RenameAlbum(ctx, albumID, title); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

func (s *DashboardService) UpdateTrack(ctx context.Context, trackID string, input libraryrepo.TrackUpdateInput) error {
	if err := s.repository.UpdateTrack(ctx, trackID, input); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

func (s *DashboardService) CreateAlbum(ctx context.Context, input libraryrepo.CreateAlbumInput) (string, error) {
	id, err := s.repository.CreateAlbum(ctx, input)
	if err != nil {
		return "", err
	}
	s.Invalidate()
	return id, nil
}

func (s *DashboardService) UpdateAlbum(ctx context.Context, albumID string, input libraryrepo.AlbumUpdateInput) error {
	if err := s.repository.UpdateAlbum(ctx, albumID, input); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

func (s *DashboardService) MergeAlbum(ctx context.Context, albumID string, input libraryrepo.AlbumMergeInput) error {
	if err := s.repository.MergeAlbum(ctx, albumID, input); err != nil {
		return err
	}
	if err := s.repository.RefreshAggregates(ctx); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

func (s *DashboardService) UpdateArtist(ctx context.Context, artistID string, input libraryrepo.ArtistUpdateInput) error {
	if err := s.repository.UpdateArtist(ctx, artistID, input); err != nil {
		return err
	}
	if err := s.repository.RefreshAggregates(ctx); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

func (s *DashboardService) DeleteArtist(ctx context.Context, artistID string) error {
	if err := s.repository.DeleteArtist(ctx, artistID); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

func (s *DashboardService) UpdateArtistCover(ctx context.Context, artistID string, data []byte, mimeType string) error {
	if s.coverSvc == nil {
		return nil
	}
	if err := s.coverSvc.SaveArtistCover(ctx, artistID, data, mimeType); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}
