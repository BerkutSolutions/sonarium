package cache

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func TestSaveOriginalAndGenerateThumb(t *testing.T) {
	t.Parallel()

	c, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	source := makePNG(t, 32, 32)
	original, err := c.SaveOriginal(source, "image/png")
	if err != nil {
		t.Fatalf("save original: %v", err)
	}
	if _, err := os.Stat(original); err != nil {
		t.Fatalf("original must exist: %v", err)
	}

	thumb, err := c.ThumbPath(original, 128)
	if err != nil {
		t.Fatalf("thumb path: %v", err)
	}
	if _, err := os.Stat(thumb); err != nil {
		t.Fatalf("thumb must exist: %v", err)
	}

	data, err := os.ReadFile(thumb)
	if err != nil {
		t.Fatalf("read thumb: %v", err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	if img.Bounds().Dx() != 128 || img.Bounds().Dy() != 128 {
		t.Fatalf("unexpected thumb size %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestPlaceholderPaths(t *testing.T) {
	t.Parallel()

	c, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	original, err := c.PlaceholderPath(0)
	if err != nil {
		t.Fatalf("placeholder original: %v", err)
	}
	if _, err := os.Stat(original); err != nil {
		t.Fatalf("placeholder original must exist: %v", err)
	}

	thumb, err := c.PlaceholderPath(256)
	if err != nil {
		t.Fatalf("placeholder thumb: %v", err)
	}
	if _, err := os.Stat(thumb); err != nil {
		t.Fatalf("placeholder thumb must exist: %v", err)
	}
}

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
