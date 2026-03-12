package metadata

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

type CoverArt struct {
	Data   []byte
	MIME   string
	Format string
}

type AudioMetadata struct {
	Artist          string
	Album           string
	Title           string
	Genre           string
	TrackNumber     int
	Year            int
	Duration        time.Duration
	Codec           string
	Bitrate         int
	ReplayGainTrack float64
	ReplayGainAlbum float64
	Cover           *CoverArt
}

type Reader struct{}

func NewReader() *Reader {
	return &Reader{}
}

func (r *Reader) Read(filePath string) (AudioMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return AudioMetadata{}, fmt.Errorf("open audio file: %w", err)
	}
	defer file.Close()

	meta, err := tag.ReadFrom(file)
	if err != nil {
		return AudioMetadata{}, fmt.Errorf("read tags: %w", err)
	}

	trackNo, _ := meta.Track()
	metadata := AudioMetadata{
		Artist:      normalizePrimaryArtist(meta.Artist()),
		Album:       strings.TrimSpace(meta.Album()),
		Title:       strings.TrimSpace(meta.Title()),
		Genre:       strings.TrimSpace(meta.Genre()),
		TrackNumber: trackNo,
		Year:        meta.Year(),
	}

	if raw, ok := meta.(interface{ Length() time.Duration }); ok {
		metadata.Duration = raw.Length()
	}
	if metadata.Duration <= 0 {
		metadata.Duration = 1 * time.Second
	}

	if audioFormat, ok := meta.(interface{ Format() tag.Format }); ok {
		metadata.Codec = string(audioFormat.Format())
	}

	if rawAudio, ok := meta.(interface{ Raw() map[string]any }); ok {
		raw := rawAudio.Raw()
		for _, key := range []string{"bitrate", "BITRATE"} {
			if value, exists := raw[key]; exists {
				switch v := value.(type) {
				case int:
					metadata.Bitrate = v
				case int64:
					metadata.Bitrate = int(v)
				case float64:
					metadata.Bitrate = int(v)
				}
				break
			}
		}
	}
	if metadata.Bitrate <= 0 {
		metadata.Bitrate = 320
	}

	if rawAudio, ok := meta.(interface{ Raw() map[string]any }); ok {
		raw := rawAudio.Raw()
		metadata.ReplayGainTrack = parseReplayGain(raw, []string{
			"REPLAYGAIN_TRACK_GAIN", "replaygain_track_gain", "rg_track_gain",
		})
		metadata.ReplayGainAlbum = parseReplayGain(raw, []string{
			"REPLAYGAIN_ALBUM_GAIN", "replaygain_album_gain", "rg_album_gain",
		})
	}

	picture := meta.Picture()
	if picture != nil && len(picture.Data) > 0 {
		metadata.Cover = &CoverArt{
			Data:   picture.Data,
			MIME:   picture.MIMEType,
			Format: picture.Type,
		}
	}

	return metadata, nil
}

func normalizePrimaryArtist(value string) string {
	artist := strings.TrimSpace(value)
	if artist == "" {
		return ""
	}

	lowered := strings.ToLower(artist)
	textSeparators := []string{
		" feat. ",
		" feat ",
		" ft. ",
		" ft ",
		" featuring ",
	}
	for _, separator := range textSeparators {
		if idx := strings.Index(lowered, separator); idx >= 0 {
			artist = artist[:idx]
			lowered = strings.ToLower(artist)
		}
	}

	symbolSeparators := []string{";", ",", "/", "&"}
	for _, separator := range symbolSeparators {
		if idx := strings.Index(artist, separator); idx >= 0 {
			artist = artist[:idx]
		}
	}

	return strings.TrimSpace(artist)
}

func parseReplayGain(raw map[string]any, keys []string) float64 {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case string:
			clean := strings.TrimSpace(strings.TrimSuffix(strings.ToLower(v), " db"))
			if clean == "" {
				continue
			}
			if parsed, err := strconv.ParseFloat(clean, 64); err == nil {
				return parsed
			}
		}
	}
	return 0
}
