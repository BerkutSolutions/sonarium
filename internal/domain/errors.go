package domain

import "errors"

var (
	ErrInvalidID          = errors.New("invalid id")
	ErrInvalidName        = errors.New("invalid name")
	ErrInvalidTitle       = errors.New("invalid title")
	ErrInvalidArtistID    = errors.New("invalid artist id")
	ErrInvalidAlbumID     = errors.New("invalid album id")
	ErrInvalidTrackID     = errors.New("invalid track id")
	ErrInvalidFilePath    = errors.New("invalid file path")
	ErrInvalidRootPath    = errors.New("invalid root path")
	ErrInvalidTrackNumber = errors.New("invalid track number")
	ErrInvalidDuration    = errors.New("invalid duration")
	ErrInvalidBitrate     = errors.New("invalid bitrate")
	ErrInvalidPosition    = errors.New("invalid playlist position")
	ErrInvalidYear        = errors.New("invalid year")
)
