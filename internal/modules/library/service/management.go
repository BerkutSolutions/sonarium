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
	ServerName      string                     `json:"server_name"`
	Version         string                     `json:"version"`
	LibraryPath     string                     `json:"library_path"`
	DatabaseOK      bool                       `json:"database_ok"`
	ScannerStatus   ScanState                  `json:"scanner_status"`
	CheckedAt       time.Time                  `json:"checked_at"`
	UpdateCheck     *appmeta.UpdateCheckResult `json:"update_check,omitempty"`
	CanCheckUpdates bool                       `json:"can_check_updates"`
}

type ManagementService struct {
	cfg       config.Config
	scanSvc   *ScanService
	dbChecker func(ctx context.Context) error
	updater   *appmeta.UpdateChecker

	mu    sync.RWMutex
	state ScanState
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
	if s.state.Status == ScanScanning {
		s.mu.Unlock()
		return false
	}
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

func (s *ManagementService) SaveUpload(ctx context.Context, fileName string, src io.Reader) (string, error) {
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
	if s.state.Status == ScanScanning {
		s.mu.Unlock()
		return "", errors.New("scanner is busy, try again in a moment")
	}
	s.state.Status = ScanScanning
	s.state.LastError = ""
	s.state.StartedAt = time.Now().UTC()
	s.state.CompletedAt = time.Time{}
	s.mu.Unlock()

	if err := os.MkdirAll(s.cfg.MusicLibraryPath, 0o755); err != nil {
		s.failScanState(err)
		return "", fmt.Errorf("create music library path: %w", err)
	}
	target := filepath.Join(s.cfg.MusicLibraryPath, baseName)
	if _, err := os.Stat(target); err == nil {
		name := strings.TrimSuffix(baseName, ext)
		target = filepath.Join(s.cfg.MusicLibraryPath, fmt.Sprintf("%s_%d%s", name, time.Now().Unix(), ext))
	}
	dst, err := os.Create(target)
	if err != nil {
		s.failScanState(err)
		return "", fmt.Errorf("create upload target: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		s.failScanState(err)
		return "", fmt.Errorf("copy upload: %w", err)
	}

	if err := s.scanSvc.ScanUploadedFile(ctx, target); err != nil {
		s.failScanState(err)
		return "", fmt.Errorf("index uploaded file: %w", err)
	}

	s.mu.Lock()
	s.state.Status = ScanCompleted
	s.state.LastError = ""
	s.state.CompletedAt = time.Now().UTC()
	s.mu.Unlock()

	return target, nil
}

func (s *ManagementService) failScanState(err error) {
	s.mu.Lock()
	s.state.Status = ScanFailed
	s.state.LastError = err.Error()
	s.state.CompletedAt = time.Now().UTC()
	s.mu.Unlock()
}

func (s *ManagementService) Settings(ctx context.Context) SettingsInfo {
	dbOK := false
	if s.dbChecker != nil {
		dbOK = s.dbChecker(ctx) == nil
	}
	return SettingsInfo{
		ServerName:    version.AppName,
		Version:       version.Version,
		LibraryPath:   s.cfg.MusicLibraryPath,
		DatabaseOK:    dbOK,
		ScannerStatus: s.ScanStatus(),
		CheckedAt:     time.Now().UTC(),
		UpdateCheck:   s.LastUpdateCheck(),
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
