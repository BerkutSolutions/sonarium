package domain

import (
	"strings"
	"time"
)

type Artist struct {
	ID        string
	Name      string
	CoverPath string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (a Artist) Validate() error {
	if strings.TrimSpace(a.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(a.Name) == "" {
		return ErrInvalidName
	}
	return nil
}
