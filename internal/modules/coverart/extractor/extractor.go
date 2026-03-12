package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
)

var directoryCoverCandidates = []string{
	"cover.jpg",
	"folder.jpg",
	"front.jpg",
	"cover.jpeg",
	"folder.jpeg",
	"front.jpeg",
	"cover.png",
	"folder.png",
	"front.png",
}

type Result struct {
	Data []byte
	MIME string
}

type Extractor struct{}

func New() *Extractor {
	return &Extractor{}
}

func (e *Extractor) Extract(trackPath string, embeddedData []byte, embeddedMIME string) (*Result, error) {
	if dirResult, ok := e.extractFromDirectory(trackPath); ok {
		return dirResult, nil
	}

	if len(embeddedData) > 0 {
		return &Result{
			Data: embeddedData,
			MIME: normalizeMIME(embeddedMIME),
		}, nil
	}

	if embeddedFromFile, ok := e.extractEmbedded(trackPath); ok {
		return embeddedFromFile, nil
	}

	return nil, nil
}

func (e *Extractor) extractFromDirectory(trackPath string) (*Result, bool) {
	dir := filepath.Dir(trackPath)
	for _, candidate := range directoryCoverCandidates {
		path := filepath.Join(dir, candidate)
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 {
			continue
		}
		return &Result{
			Data: data,
			MIME: mimeByExt(path),
		}, true
	}
	return nil, false
}

func (e *Extractor) extractEmbedded(trackPath string) (*Result, bool) {
	ext := strings.ToLower(filepath.Ext(trackPath))
	if ext != ".mp3" && ext != ".flac" {
		return nil, false
	}

	file, err := os.Open(trackPath)
	if err != nil {
		return nil, false
	}
	defer file.Close()

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return nil, false
	}
	picture := metadata.Picture()
	if picture == nil || len(picture.Data) == 0 {
		return nil, false
	}

	return &Result{
		Data: picture.Data,
		MIME: normalizeMIME(picture.MIMEType),
	}, true
}

func normalizeMIME(m string) string {
	if strings.TrimSpace(m) == "" {
		return "image/jpeg"
	}
	return m
}

func mimeByExt(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

func (e *Extractor) ExtractFromDirectory(trackPath string) (*Result, error) {
	result, ok := e.extractFromDirectory(trackPath)
	if !ok {
		return nil, nil
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("empty cover data")
	}
	return result, nil
}
