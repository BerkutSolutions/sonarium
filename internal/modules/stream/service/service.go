package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"music-server/internal/domain"
)

var (
	ErrTrackNotFound = errors.New("track not found")
	ErrFileNotFound  = errors.New("track file not found")
)

type StreamableTrack struct {
	TrackID   string
	FilePath  string
	ModTime   time.Time
	MIMEType  string
	FileName  string
	FileSize  int64
	CodecHint string
}

type Service struct {
	trackRepository domain.TrackRepository
}

func New(trackRepository domain.TrackRepository) *Service {
	return &Service{trackRepository: trackRepository}
}

func (s *Service) ResolveTrack(ctx context.Context, trackID string) (StreamableTrack, error) {
	track, err := s.trackRepository.GetByID(ctx, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StreamableTrack{}, ErrTrackNotFound
		}
		return StreamableTrack{}, fmt.Errorf("get track by id: %w", err)
	}

	info, err := os.Stat(track.FilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return StreamableTrack{}, ErrFileNotFound
		}
		return StreamableTrack{}, fmt.Errorf("stat track file: %w", err)
	}
	if info.IsDir() {
		return StreamableTrack{}, ErrFileNotFound
	}

	return StreamableTrack{
		TrackID:   track.ID,
		FilePath:  track.FilePath,
		ModTime:   info.ModTime().UTC(),
		MIMEType:  mimeTypeFromPath(track.FilePath),
		FileName:  filepath.Base(track.FilePath),
		FileSize:  info.Size(),
		CodecHint: track.Codec,
	}, nil
}

func mimeTypeFromPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	default:
		return "application/octet-stream"
	}
}
