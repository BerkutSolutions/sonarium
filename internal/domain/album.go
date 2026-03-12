package domain

import (
	"strings"
	"time"
)

type Album struct {
	ID        string
	Title     string
	ArtistID  string
	Year      int
	CoverPath string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (a Album) Validate() error {
	if strings.TrimSpace(a.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(a.Title) == "" {
		return ErrInvalidTitle
	}
	if strings.TrimSpace(a.ArtistID) == "" {
		return ErrInvalidArtistID
	}
	if a.Year < 0 || a.Year > 3000 {
		return ErrInvalidYear
	}
	return nil
}
