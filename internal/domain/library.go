package domain

import (
	"path/filepath"
	"strings"
	"time"
)

type Library struct {
	RootPath   string
	LastScanAt time.Time
}

func (l Library) Validate() error {
	if strings.TrimSpace(l.RootPath) == "" {
		return ErrInvalidRootPath
	}
	return nil
}

func (l Library) NormalizedRootPath() string {
	if strings.TrimSpace(l.RootPath) == "" {
		return ""
	}
	return filepath.Clean(l.RootPath)
}
