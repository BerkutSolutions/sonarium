package service

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"music-server/internal/domain"
	coverservice "music-server/internal/modules/coverart/service"
	"music-server/internal/modules/library/metadata"
	libraryrepo "music-server/internal/modules/library/repository"
	"music-server/internal/modules/library/scanner"
	loudnessservice "music-server/internal/modules/loudness/service"
	waveformservice "music-server/internal/modules/waveform/service"
	"music-server/internal/platform/config"
	"music-server/internal/storage/repositories"
)

type ScanService struct {
	cfg               config.Config
	logger            *zap.Logger
	scanner           *scanner.FilesystemScanner
	metadataReader    *metadata.Reader
	artistRepository  domain.ArtistRepository
	albumRepository   domain.AlbumRepository
	trackRepository   domain.TrackRepository
	libraryRepository domain.LibraryRepository
	fingerprintRepo   *repositories.FileFingerprintRepository
	coverService      *coverservice.Service
	loudnessService   *loudnessservice.Service
	waveformService   *waveformservice.Service
	statsRepository   *libraryrepo.Repository
	artistCacheByName map[string]domain.Artist
	albumCacheByKey   map[string]domain.Album
	cacheMu           sync.RWMutex
}

type scanJob struct {
	entry scanner.FileEntry
}

type scanResult struct {
	entry    scanner.FileEntry
	metadata metadata.AudioMetadata
	err      error
}

func NewScanService(
	cfg config.Config,
	logger *zap.Logger,
	scanner *scanner.FilesystemScanner,
	metadataReader *metadata.Reader,
	artistRepository domain.ArtistRepository,
	albumRepository domain.AlbumRepository,
	trackRepository domain.TrackRepository,
	libraryRepository domain.LibraryRepository,
	fingerprintRepo *repositories.FileFingerprintRepository,
	coverService *coverservice.Service,
	loudnessService *loudnessservice.Service,
	waveformService *waveformservice.Service,
	statsRepository *libraryrepo.Repository,
) *ScanService {
	return &ScanService{
		cfg:               cfg,
		logger:            logger,
		scanner:           scanner,
		metadataReader:    metadataReader,
		artistRepository:  artistRepository,
		albumRepository:   albumRepository,
		trackRepository:   trackRepository,
		libraryRepository: libraryRepository,
		fingerprintRepo:   fingerprintRepo,
		coverService:      coverService,
		loudnessService:   loudnessService,
		waveformService:   waveformService,
		statsRepository:   statsRepository,
		artistCacheByName: make(map[string]domain.Artist),
		albumCacheByKey:   make(map[string]domain.Album),
	}
}

func (s *ScanService) Scan(ctx context.Context) error {
	s.logger.Info("library scan started", zap.String("music_root", s.cfg.MusicLibraryPath))

	libraryState, err := s.libraryRepository.Get(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("get library state: %w", err)
	}
	if libraryState.RootPath != s.cfg.MusicLibraryPath {
		saveErr := s.libraryRepository.Save(ctx, domain.Library{
			RootPath:   s.cfg.MusicLibraryPath,
			LastScanAt: libraryState.LastScanAt,
		})
		if saveErr != nil {
			return fmt.Errorf("save library root path: %w", saveErr)
		}
	}

	fingerprints, err := s.fingerprintRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("load fingerprints: %w", err)
	}
	fingerprintIndex := make(map[string]repositories.FileFingerprint, len(fingerprints))
	for _, fp := range fingerprints {
		fingerprintIndex[fp.FilePath] = fp
	}

	files, err := s.scanner.Scan(s.cfg.MusicLibraryPath)
	if err != nil {
		return fmt.Errorf("scan filesystem: %w", err)
	}

	jobs := make(chan scanJob)
	results := make(chan scanResult)
	workers := s.cfg.ScannerWorkers
	if workers <= 0 {
		workers = 1
	}

	var workerWG sync.WaitGroup
	for i := 0; i < workers; i++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			for job := range jobs {
				meta, err := s.metadataReader.Read(job.entry.Path)
				results <- scanResult{
					entry:    job.entry,
					metadata: meta,
					err:      err,
				}
			}
		}()
	}

	filesToProcess := 0
	go func() {
		defer close(jobs)
		for _, file := range files {
			if isUnchanged(file, fingerprintIndex[file.Path]) {
				continue
			}
			filesToProcess++
			jobs <- scanJob{entry: file}
		}
	}()

	go func() {
		workerWG.Wait()
		close(results)
	}()

	processed := 0
	skipped := len(files) - filesToProcess

	for result := range results {
		if result.err != nil {
			s.logger.Warn("metadata read failed",
				zap.String("file", result.entry.Path),
				zap.Error(result.err),
			)
			result.metadata = fallbackMetadata(result.entry.Path)
		}

		if err := s.persistTrack(ctx, result.entry, result.metadata); err != nil {
			s.logger.Warn("persist track failed",
				zap.String("file", result.entry.Path),
				zap.Error(err),
			)
			continue
		}

		if err := s.fingerprintRepo.Upsert(
			ctx,
			result.entry.Path,
			result.entry.Size,
			result.entry.ModTime,
			fingerprintHash(result.entry),
		); err != nil {
			s.logger.Warn("fingerprint upsert failed",
				zap.String("file", result.entry.Path),
				zap.Error(err),
			)
		}
		processed++
	}

	if err := s.libraryRepository.UpdateLastScanAt(ctx); err != nil {
		return fmt.Errorf("update library last_scan_at: %w", err)
	}
	if s.statsRepository != nil {
		if err := s.statsRepository.RefreshAggregates(ctx); err != nil {
			s.logger.Warn("failed to refresh library aggregates", zap.Error(err))
		}
	}

	s.logger.Info("library scan completed",
		zap.Int("total_files", len(files)),
		zap.Int("processed", processed),
		zap.Int("skipped", skipped),
	)

	return nil
}

func (s *ScanService) persistTrack(ctx context.Context, entry scanner.FileEntry, meta metadata.AudioMetadata) error {
	artistName := fallback(meta.Artist, "Unknown Artist")
	albumTitle := fallback(meta.Album, "Unknown Album")
	trackTitle := fallback(meta.Title, strings.TrimSuffix(filepath.Base(entry.Path), filepath.Ext(entry.Path)))

	artist, err := s.ensureArtist(ctx, artistName)
	if err != nil {
		return err
	}

	album, err := s.ensureAlbum(ctx, artist, albumTitle, meta.Year, "")
	if err != nil {
		return err
	}

	if s.coverService != nil {
		updatedArtist, updatedAlbum, coverErr := s.coverService.ResolveAndAttach(
			ctx,
			artist,
			album,
			entry.Path,
			extractCoverData(meta),
			extractCoverMIME(meta),
		)
		if coverErr != nil {
			s.logger.Warn("cover resolve failed", zap.Error(coverErr), zap.String("file", entry.Path))
		} else {
			artist = updatedArtist
			album = updatedAlbum
		}
	}

	existingTrack, err := s.trackRepository.GetByFilePath(ctx, entry.Path)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("find track by file path: %w", err)
	}

	gains := loudnessservice.Gains{
		Track: meta.ReplayGainTrack,
		Album: meta.ReplayGainAlbum,
	}
	if s.loudnessService != nil {
		gains = s.loudnessService.Resolve(meta)
	}

	track := domain.Track{
		Title:           trackTitle,
		AlbumID:         album.ID,
		ArtistID:        artist.ID,
		TrackNumber:     fallbackTrackNumber(meta.TrackNumber),
		Duration:        meta.Duration,
		FilePath:        entry.Path,
		Genre:           strings.TrimSpace(meta.Genre),
		Codec:           fallback(meta.Codec, strings.TrimPrefix(strings.ToLower(filepath.Ext(entry.Path)), ".")),
		Bitrate:         fallbackBitrate(meta.Bitrate),
		ReplayGainTrack: gains.Track,
		ReplayGainAlbum: gains.Album,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	if existingTrack.ID != "" {
		track.ID = existingTrack.ID
		track.CreatedAt = existingTrack.CreatedAt
	}

	if err := s.trackRepository.Upsert(ctx, track); err != nil {
		return fmt.Errorf("upsert track: %w", err)
	}
	if s.waveformService != nil {
		if err := s.waveformService.GenerateForFile(entry.Path); err != nil {
			s.logger.Warn("waveform generation failed", zap.String("file", entry.Path), zap.Error(err))
		}
	}

	return nil
}

func (s *ScanService) ensureArtist(ctx context.Context, name string) (domain.Artist, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	s.cacheMu.RLock()
	cached, ok := s.artistCacheByName[key]
	s.cacheMu.RUnlock()
	if ok {
		return cached, nil
	}

	artist, err := s.artistRepository.GetByName(ctx, name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.Artist{}, fmt.Errorf("get artist by name: %w", err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		artist = domain.Artist{
			Name:      name,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err := s.artistRepository.Upsert(ctx, artist); err != nil {
			return domain.Artist{}, fmt.Errorf("upsert artist: %w", err)
		}
		artist, err = s.artistRepository.GetByName(ctx, name)
		if err != nil {
			return domain.Artist{}, fmt.Errorf("reload artist by name: %w", err)
		}
	}

	s.cacheMu.Lock()
	s.artistCacheByName[key] = artist
	s.cacheMu.Unlock()
	return artist, nil
}

func (s *ScanService) ensureAlbum(ctx context.Context, artist domain.Artist, title string, year int, coverPath string) (domain.Album, error) {
	key := strings.ToLower(strings.TrimSpace(artist.ID + "::" + title))
	s.cacheMu.RLock()
	cached, ok := s.albumCacheByKey[key]
	s.cacheMu.RUnlock()
	if ok {
		return cached, nil
	}

	album, err := s.albumRepository.GetByTitleAndArtistID(ctx, title, artist.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.Album{}, fmt.Errorf("get album by title and artist: %w", err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		album = domain.Album{
			Title:     title,
			ArtistID:  artist.ID,
			Year:      fallbackYear(year),
			CoverPath: coverPath,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err := s.albumRepository.Upsert(ctx, album); err != nil {
			return domain.Album{}, fmt.Errorf("upsert album: %w", err)
		}
		album, err = s.albumRepository.GetByTitleAndArtistID(ctx, title, artist.ID)
		if err != nil {
			return domain.Album{}, fmt.Errorf("reload album by title and artist: %w", err)
		}
	} else if coverPath != "" && album.CoverPath == "" {
		album.CoverPath = coverPath
		album.UpdatedAt = time.Now().UTC()
		if err := s.albumRepository.Upsert(ctx, album); err != nil {
			return domain.Album{}, fmt.Errorf("update album cover: %w", err)
		}
	}

	s.cacheMu.Lock()
	s.albumCacheByKey[key] = album
	s.cacheMu.Unlock()
	return album, nil
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func fallbackTrackNumber(number int) int {
	if number <= 0 {
		return 1
	}
	return number
}

func fallbackYear(year int) int {
	if year < 0 || year > 3000 {
		return 0
	}
	return year
}

func fallbackBitrate(bitrate int) int {
	if bitrate <= 0 {
		return 320
	}
	return bitrate
}

func isUnchanged(file scanner.FileEntry, stored repositories.FileFingerprint) bool {
	if stored.FilePath == "" {
		return false
	}
	if stored.FileSize != file.Size || !stored.ModTime.Equal(file.ModTime) {
		return false
	}
	return stored.FingerprintHash == fingerprintHash(file)
}

func extractCoverData(meta metadata.AudioMetadata) []byte {
	if meta.Cover == nil {
		return nil
	}
	return meta.Cover.Data
}

func extractCoverMIME(meta metadata.AudioMetadata) string {
	if meta.Cover == nil {
		return ""
	}
	return meta.Cover.MIME
}

func fallbackMetadata(path string) metadata.AudioMetadata {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return metadata.AudioMetadata{
		Artist:      "Unknown Artist",
		Album:       "Unknown Album",
		Title:       base,
		Genre:       "",
		TrackNumber: 1,
		Year:        0,
		Duration:    time.Second,
		Codec:       strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
		Bitrate:     320,
	}
}

func fingerprintHash(file scanner.FileEntry) string {
	input := fmt.Sprintf("%s|%d|%d", file.Path, file.Size, file.ModTime.UnixNano())
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func (s *ScanService) ScanUploadedFile(ctx context.Context, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat uploaded file: %w", err)
	}
	entry := scanner.FileEntry{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime().UTC(),
	}

	meta, err := s.metadataReader.Read(entry.Path)
	if err != nil {
		s.logger.Warn("metadata read failed for uploaded file", zap.String("file", entry.Path), zap.Error(err))
		meta = fallbackMetadata(entry.Path)
	}
	if err := s.persistTrack(ctx, entry, meta); err != nil {
		return fmt.Errorf("persist uploaded track: %w", err)
	}
	if err := s.fingerprintRepo.Upsert(ctx, entry.Path, entry.Size, entry.ModTime, fingerprintHash(entry)); err != nil {
		return fmt.Errorf("save uploaded fingerprint: %w", err)
	}
	if err := s.libraryRepository.UpdateLastScanAt(ctx); err != nil {
		return fmt.Errorf("update last scan: %w", err)
	}
	if s.statsRepository != nil {
		if err := s.statsRepository.RefreshAggregates(ctx); err != nil {
			s.logger.Warn("failed to refresh aggregates after upload", zap.Error(err))
		}
	}
	return nil
}
