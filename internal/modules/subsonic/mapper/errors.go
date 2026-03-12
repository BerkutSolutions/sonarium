package mapper

import (
	"database/sql"
	"errors"
	"os"

	streamservice "music-server/internal/modules/stream/service"
	subsonicservice "music-server/internal/modules/subsonic/service"
)

func ErrorFrom(err error) (int, string) {
	switch {
	case errors.Is(err, subsonicservice.ErrMissingAuthParams):
		return 10, "missing authentication parameters"
	case errors.Is(err, subsonicservice.ErrMissingProtocolParams):
		return 10, "missing required protocol parameters"
	case errors.Is(err, subsonicservice.ErrInvalidCredentials):
		return 40, "wrong username or password"
	case errors.Is(err, subsonicservice.ErrUnsupportedAPIVersion):
		return 20, "client must upgrade"
	case errors.Is(err, subsonicservice.ErrUnsupportedFormatParam):
		return 0, "unsupported response format"
	case errors.Is(err, sql.ErrNoRows):
		return 70, "requested data not found"
	case errors.Is(err, streamservice.ErrTrackNotFound):
		return 70, "track not found"
	case errors.Is(err, streamservice.ErrFileNotFound), errors.Is(err, os.ErrNotExist):
		return 70, "media file not found"
	case errors.Is(err, subsonicservice.ErrCoverArtNotFound):
		return 70, "cover art not found"
	default:
		return 0, "internal server error"
	}
}
