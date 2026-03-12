package extractor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPrefersDirectoryCover(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	trackPath := filepath.Join(dir, "track.mp3")
	coverPath := filepath.Join(dir, "cover.jpg")

	if err := os.WriteFile(trackPath, []byte("not-audio"), 0o644); err != nil {
		t.Fatalf("write track fixture: %v", err)
	}
	expected := []byte{0x01, 0x02, 0x03, 0x04}
	if err := os.WriteFile(coverPath, expected, 0o644); err != nil {
		t.Fatalf("write cover fixture: %v", err)
	}

	result, err := New().Extract(trackPath, nil, "")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil extraction result")
	}
	if result.MIME != "image/jpeg" {
		t.Fatalf("expected image/jpeg, got %s", result.MIME)
	}
	if string(result.Data) != string(expected) {
		t.Fatalf("unexpected data size/content")
	}
}

func TestExtractUsesEmbeddedWhenDirectoryCoverMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	trackPath := filepath.Join(dir, "track.flac")
	if err := os.WriteFile(trackPath, []byte("not-audio"), 0o644); err != nil {
		t.Fatalf("write track fixture: %v", err)
	}

	embedded := []byte{0x0A, 0x0B, 0x0C}
	result, err := New().Extract(trackPath, embedded, "image/png")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil extraction result")
	}
	if result.MIME != "image/png" {
		t.Fatalf("expected image/png, got %s", result.MIME)
	}
	if string(result.Data) != string(embedded) {
		t.Fatalf("unexpected embedded data")
	}
}
