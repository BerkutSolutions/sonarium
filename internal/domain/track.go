package domain

import (
	"strings"
	"time"
)

type Track struct {
	ID              string
	Title           string
	AlbumID         string
	ArtistID        string
	TrackNumber     int
	Duration        time.Duration
	FilePath        string
	Genre           string
	Codec           string
	Bitrate         int
	ReplayGainTrack float64
	ReplayGainAlbum float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (t Track) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(t.Title) == "" {
		return ErrInvalidTitle
	}
	if strings.TrimSpace(t.AlbumID) == "" {
		return ErrInvalidAlbumID
	}
	if strings.TrimSpace(t.ArtistID) == "" {
		return ErrInvalidArtistID
	}
	if t.TrackNumber <= 0 {
		return ErrInvalidTrackNumber
	}
	if t.Duration <= 0 {
		return ErrInvalidDuration
	}
	if strings.TrimSpace(t.FilePath) == "" {
		return ErrInvalidFilePath
	}
	if t.Bitrate <= 0 {
		return ErrInvalidBitrate
	}
	return nil
}
