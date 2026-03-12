package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"music-server/internal/domain"
)

type Service struct {
	dataPath        string
	trackRepository domain.TrackRepository
}

func New(dataPath string, trackRepository domain.TrackRepository) *Service {
	return &Service{
		dataPath:        dataPath,
		trackRepository: trackRepository,
	}
}

func (s *Service) GenerateForFile(filePath string) error {
	if filePath == "" {
		return nil
	}
	target := s.waveformPath(filePath)
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	values, err := computeWaveform(filePath, 192)
	if err != nil {
		return err
	}
	return s.writeWaveform(target, values)
}

func (s *Service) GetForTrack(ctx context.Context, trackID string) ([]int, error) {
	track, err := s.trackRepository.GetByID(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("get track: %w", err)
	}
	target := s.waveformPath(track.FilePath)
	if _, err := os.Stat(target); err != nil {
		if err := s.GenerateForFile(track.FilePath); err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return nil, fmt.Errorf("read waveform: %w", err)
	}
	var values []int
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("decode waveform json: %w", err)
	}
	return values, nil
}

func (s *Service) waveformPath(filePath string) string {
	hash := sha1.Sum([]byte(filePath))
	name := hex.EncodeToString(hash[:]) + ".json"
	return filepath.Join(s.dataPath, "waveforms", name)
}

func (s *Service) writeWaveform(target string, values []int) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create waveform dir: %w", err)
	}
	data, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal waveform: %w", err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("write waveform: %w", err)
	}
	return nil
}

func computeWaveform(path string, buckets int) ([]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	if info.Size() <= 0 || buckets <= 0 {
		return make([]int, 0), nil
	}

	chunkSize := info.Size() / int64(buckets)
	if chunkSize < 1024 {
		chunkSize = 1024
	}

	values := make([]int, 0, buckets)
	buf := make([]byte, chunkSize)
	for len(values) < buckets {
		n, readErr := io.ReadFull(file, buf)
		if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
			return nil, fmt.Errorf("read file chunk: %w", readErr)
		}
		if n <= 0 {
			break
		}
		total := 0
		for i := 0; i < n; i++ {
			b := int(buf[i])
			if b > 127 {
				b = 255 - b
			}
			total += b
		}
		avg := total / n
		if avg > 100 {
			avg = 100
		}
		values = append(values, avg)
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
	}
	for len(values) < buckets {
		values = append(values, 0)
	}
	return values, nil
}
