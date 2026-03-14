package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"music-server/internal/appmeta"
	libraryrepo "music-server/internal/modules/library/repository"
	"music-server/internal/platform/config"
	"music-server/pkg/version"
)

type ScanStatus string

const (
	ScanIdle      ScanStatus = "Idle"
	ScanScanning  ScanStatus = "Scanning"
	ScanCompleted ScanStatus = "Completed"
	ScanFailed    ScanStatus = "Failed"
)

type ScanState struct {
	Status      ScanStatus `json:"status"`
	LastError   string     `json:"last_error,omitempty"`
	StartedAt   time.Time  `json:"started_at,omitempty"`
	CompletedAt time.Time  `json:"completed_at,omitempty"`
}

type SettingsInfo struct {
	ServerName        string                     `json:"server_name"`
	Version           string                     `json:"version"`
	LibraryPath       string                     `json:"library_path"`
	DatabaseOK        bool                       `json:"database_ok"`
	ScannerStatus     ScanState                  `json:"scanner_status"`
	UploadConcurrency int                        `json:"upload_concurrency"`
	AutoCheckUpdates  bool                       `json:"auto_check_updates"`
	CheckedAt         time.Time                  `json:"checked_at"`
	UpdateCheck       *appmeta.UpdateCheckResult `json:"update_check,omitempty"`
	CanCheckUpdates   bool                       `json:"can_check_updates"`
}

type ManagementService struct {
	cfg       config.Config
	scanSvc   *ScanService
	dbChecker func(ctx context.Context) error
	updater   *appmeta.UpdateChecker

	mu              sync.RWMutex
	state           ScanState
	fullScanRunning bool
	pendingUploads  int
}

func NewManagementService(cfg config.Config, scanSvc *ScanService, dbChecker func(ctx context.Context) error, updater *appmeta.UpdateChecker) *ManagementService {
	return &ManagementService{
		cfg:       cfg,
		scanSvc:   scanSvc,
		dbChecker: dbChecker,
		updater:   updater,
		state: ScanState{
			Status: ScanIdle,
		},
	}
}

func (s *ManagementService) LibraryPath() string {
	return s.cfg.MusicLibraryPath
}

func (s *ManagementService) ScanStatus() ScanState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *ManagementService) TriggerScan() bool {
	s.mu.Lock()
	if s.state.Status == ScanScanning || s.fullScanRunning || s.pendingUploads > 0 {
		s.mu.Unlock()
		return false
	}
	s.fullScanRunning = true
	s.state.Status = ScanScanning
	s.state.LastError = ""
	s.state.StartedAt = time.Now().UTC()
	s.state.CompletedAt = time.Time{}
	s.mu.Unlock()

	go s.runScan()
	return true
}

func (s *ManagementService) runScan() {
	err := s.scanSvc.Scan(context.Background())
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fullScanRunning = false
	if s.pendingUploads > 0 {
		if err != nil {
			s.state.LastError = err.Error()
		}
		s.state.Status = ScanScanning
		return
	}
	if err != nil {
		s.state.Status = ScanFailed
		s.state.LastError = err.Error()
		s.state.CompletedAt = time.Now().UTC()
		return
	}
	s.state.Status = ScanCompleted
	s.state.LastError = ""
	s.state.CompletedAt = time.Now().UTC()
}

func (s *ManagementService) SaveUpload(ctx context.Context, userID, fileName string, src io.Reader, skipDuplicates bool) (string, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".mp3", ".flac", ".ogg", ".m4a":
	default:
		return "", errors.New("unsupported file type")
	}
	baseName := filepath.Base(fileName)
	if baseName == "." || baseName == string(filepath.Separator) || baseName == "" {
		return "", errors.New("invalid file name")
	}

	s.mu.Lock()
	if s.fullScanRunning {
		s.mu.Unlock()
		return "", errors.New("scanner is busy, try again in a moment")
	}
	s.pendingUploads++
	s.state.Status = ScanScanning
	s.state.LastError = ""
	if s.state.StartedAt.IsZero() {
		s.state.StartedAt = time.Now().UTC()
	}
	s.state.CompletedAt = time.Time{}
	s.mu.Unlock()

	if err := os.MkdirAll(s.cfg.MusicLibraryPath, 0o755); err != nil {
		s.completeUpload(err)
		return "", fmt.Errorf("create music library path: %w", err)
	}
	target := filepath.Join(s.cfg.MusicLibraryPath, baseName)
	if _, err := os.Stat(target); err == nil {
		name := strings.TrimSuffix(baseName, ext)
		target = filepath.Join(s.cfg.MusicLibraryPath, fmt.Sprintf("%s_%d%s", name, time.Now().UnixNano(), ext))
	}
	dst, err := os.Create(target)
	if err != nil {
		s.completeUpload(err)
		return "", fmt.Errorf("create upload target: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		s.completeUpload(err)
		return "", fmt.Errorf("copy upload: %w", err)
	}

	if err := s.scanSvc.ScanUploadedFile(ctx, target, userID, skipDuplicates); err != nil {
		if errors.Is(err, ErrDuplicateUpload) {
			_ = os.Remove(target)
			s.completeUpload(nil)
			return "", ErrDuplicateUpload
		}
		s.completeUpload(err)
		return "", fmt.Errorf("index uploaded file: %w", err)
	}

	s.completeUpload(nil)
	return target, nil
}

func (s *ManagementService) completeUpload(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pendingUploads > 0 {
		s.pendingUploads--
	}
	if err != nil {
		s.state.LastError = err.Error()
	}
	if s.fullScanRunning || s.pendingUploads > 0 {
		s.state.Status = ScanScanning
		return
	}
	if s.state.LastError != "" {
		s.state.Status = ScanFailed
	} else {
		s.state.Status = ScanCompleted
	}
	s.state.CompletedAt = time.Now().UTC()
}

func (s *ManagementService) Settings(ctx context.Context) SettingsInfo {
	dbOK := false
	if s.dbChecker != nil {
		dbOK = s.dbChecker(ctx) == nil
	}
	concurrency := 4
	autoCheckUpdates := true
	if s.scanSvc != nil && s.scanSvc.statsRepository != nil {
		if value, err := s.scanSvc.statsRepository.UploadConcurrency(ctx); err == nil {
			concurrency = value
		}
		if value, err := s.scanSvc.statsRepository.AutoCheckUpdates(ctx); err == nil {
			autoCheckUpdates = value
		}
	}
	return SettingsInfo{
		ServerName:        version.AppName,
		Version:           version.Version,
		LibraryPath:       s.cfg.MusicLibraryPath,
		DatabaseOK:        dbOK,
		ScannerStatus:     s.ScanStatus(),
		UploadConcurrency: concurrency,
		AutoCheckUpdates:  autoCheckUpdates,
		CheckedAt:         time.Now().UTC(),
		UpdateCheck:       s.LastUpdateCheck(),
	}
}

func (s *ManagementService) CheckUpdates(ctx context.Context) (*appmeta.UpdateCheckResult, error) {
	if s.updater == nil {
		return nil, nil
	}
	return s.updater.Check(ctx, version.Version)
}

func (s *ManagementService) LastUpdateCheck() *appmeta.UpdateCheckResult {
	if s.updater == nil {
		return nil
	}
	return s.updater.LastResult()
}

func (s *ManagementService) StorageUsage(ctx context.Context) (libraryrepo.StorageUsage, error) {
	if s.scanSvc == nil || s.scanSvc.statsRepository == nil {
		return libraryrepo.StorageUsage{}, nil
	}
	return s.scanSvc.statsRepository.StorageUsage(ctx)
}

func (s *ManagementService) DeleteAllMusic(ctx context.Context) error {
	if s.scanSvc == nil || s.scanSvc.statsRepository == nil {
		return nil
	}
	if err := s.scanSvc.statsRepository.ClearLibrary(ctx); err != nil {
		return err
	}
	s.scanSvc.ResetCaches()
	if err := clearDirectoryContents(s.cfg.MusicLibraryPath); err != nil {
		return err
	}
	if err := clearDirectoryContents(filepath.Join(s.cfg.DataPath, "covers")); err != nil {
		return err
	}
	if err := clearDirectoryContents(filepath.Join(s.cfg.DataPath, "waveforms")); err != nil {
		return err
	}
	s.mu.Lock()
	s.pendingUploads = 0
	s.fullScanRunning = false
	s.state.Status = ScanCompleted
	s.state.LastError = ""
	s.state.CompletedAt = time.Now().UTC()
	s.mu.Unlock()
	return nil
}

func (s *ManagementService) SetUploadConcurrency(ctx context.Context, value int) error {
	if s.scanSvc == nil || s.scanSvc.statsRepository == nil {
		return nil
	}
	return s.scanSvc.statsRepository.SetUploadConcurrency(ctx, value)
}

func (s *ManagementService) SetAutoCheckUpdates(ctx context.Context, value bool) error {
	if s.scanSvc == nil || s.scanSvc.statsRepository == nil {
		return nil
	}
	return s.scanSvc.statsRepository.SetAutoCheckUpdates(ctx, value)
}

func clearDirectoryContents(root string) error {
	if strings.TrimSpace(root) == "" {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read dir %s: %w", root, err)
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return fmt.Errorf("remove %s: %w", filepath.Join(root, entry.Name()), err)
		}
	}
	return nil
}
