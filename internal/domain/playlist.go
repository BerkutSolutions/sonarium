package domain

import (
	"strings"
	"time"
)

type Playlist struct {
	ID          string
	Name        string
	Description string
	OwnerUserID string
	AccessRole  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PlaylistTrack struct {
	PlaylistID string
	TrackID    string
	Position   int
}

func (p Playlist) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(p.Name) == "" {
		return ErrInvalidName
	}
	return nil
}

func (pt PlaylistTrack) Validate() error {
	if strings.TrimSpace(pt.PlaylistID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(pt.TrackID) == "" {
		return ErrInvalidTrackID
	}
	if pt.Position <= 0 {
		return ErrInvalidPosition
	}
	return nil
}
