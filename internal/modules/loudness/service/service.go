package service

import (
	"math"

	"music-server/internal/modules/library/metadata"
)

type Gains struct {
	Track float64
	Album float64
}

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Resolve(meta metadata.AudioMetadata) Gains {
	track := meta.ReplayGainTrack
	album := meta.ReplayGainAlbum
	if track == 0 {
		track = heuristicTrackGain(meta.Bitrate)
	}
	if album == 0 {
		album = track
	}
	return Gains{
		Track: round(track, 2),
		Album: round(album, 2),
	}
}

func heuristicTrackGain(bitrate int) float64 {
	if bitrate <= 0 {
		return 0
	}
	if bitrate >= 320 {
		return -6
	}
	if bitrate >= 256 {
		return -5
	}
	if bitrate >= 192 {
		return -4
	}
	return -3
}

func round(value float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(value*pow) / pow
}

