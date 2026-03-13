package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
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
		Album:       normalizeTagText(meta.Album()),
		Title:       normalizeTagText(meta.Title()),
		Genre:       normalizeTagText(meta.Genre()),
		TrackNumber: trackNo,
		Year:        meta.Year(),
	}

	if raw, ok := meta.(interface{ Length() time.Duration }); ok {
		metadata.Duration = raw.Length()
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
		metadata.Bitrate = 0
	}

	if metadata.Duration <= time.Second || metadata.Bitrate <= 0 {
		r.enrichWithFFprobe(filePath, &metadata)
	}

	if metadata.Duration <= 0 {
		metadata.Duration = 1 * time.Second
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
	artist := normalizeTagText(value)
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

func normalizeTagText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
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

func (r *Reader) enrichWithFFprobe(filePath string, metadata *AudioMetadata) {
	if metadata == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration,bit_rate",
		"-of", "json",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return
	}

	var payload struct {
		Format struct {
			Duration string `json:"duration"`
			BitRate  string `json:"bit_rate"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return
	}

	if metadata.Duration <= 0 {
		if seconds, err := strconv.ParseFloat(strings.TrimSpace(payload.Format.Duration), 64); err == nil && seconds > 0 {
			metadata.Duration = time.Duration(math.Round(seconds * float64(time.Second)))
		}
	}
	if metadata.Bitrate <= 0 {
		if bitrate, err := strconv.Atoi(strings.TrimSpace(payload.Format.BitRate)); err == nil && bitrate > 0 {
			metadata.Bitrate = bitrate / 1000
		}
	}
}
