package cache

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

var SupportedThumbSizes = map[int]struct{}{
	64:  {},
	128: {},
	256: {},
}

type Cache struct {
	rootPath     string
	originalPath string
	thumbPath    string
}

func New(dataPath string) (*Cache, error) {
	root := filepath.Join(dataPath, "covers")
	original := filepath.Join(root, "original")
	thumb := filepath.Join(root, "thumb")

	cache := &Cache{
		rootPath:     root,
		originalPath: original,
		thumbPath:    thumb,
	}
	if err := cache.ensureLayout(); err != nil {
		return nil, err
	}

	return cache, nil
}

func (c *Cache) SaveOriginal(data []byte, mimeType string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty cover data")
	}
	if err := c.ensureLayout(); err != nil {
		return "", err
	}
	ext := extByMIME(mimeType)
	hash := sha1.Sum(data)
	fileName := hex.EncodeToString(hash[:]) + ext
	path := filepath.Join(c.originalPath, fileName)

	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := writeFileAtomically(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write original cover: %w", err)
	}
	return path, nil
}

func (c *Cache) ThumbPath(originalPath string, size int) (string, error) {
	if _, ok := SupportedThumbSizes[size]; !ok {
		return "", fmt.Errorf("unsupported thumb size %d", size)
	}
	if err := c.ensureLayout(); err != nil {
		return "", err
	}
	if strings.TrimSpace(originalPath) == "" {
		return c.PlaceholderPath(size)
	}
	if _, err := os.Stat(originalPath); err != nil {
		return c.PlaceholderPath(size)
	}
	fileName := strings.TrimSuffix(filepath.Base(originalPath), filepath.Ext(originalPath)) + ".jpg"
	dir := filepath.Join(c.thumbPath, fmt.Sprintf("%d", size))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create thumb dir: %w", err)
	}
	target := filepath.Join(dir, fileName)
	if _, err := os.Stat(target); err == nil {
		return target, nil
	}

	originalData, err := os.ReadFile(originalPath)
	if err != nil {
		return c.PlaceholderPath(size)
	}
	decoded, _, err := image.DecodeConfig(bytes.NewReader(originalData))
	if err != nil {
		return c.PlaceholderPath(size)
	}
	if decoded.Width == 0 || decoded.Height == 0 {
		return c.PlaceholderPath(size)
	}

	img, _, err := image.Decode(bytes.NewReader(originalData))
	if err != nil {
		return c.PlaceholderPath(size)
	}
	thumb := resizeNearest(img, size, size)

	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, thumb, &jpeg.Options{Quality: 85}); err != nil {
		return "", fmt.Errorf("encode thumb jpeg: %w", err)
	}
	if err := writeFileAtomically(target, encoded.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("write thumb file: %w", err)
	}
	return target, nil
}

func (c *Cache) PlaceholderPath(size int) (string, error) {
	if err := c.ensureLayout(); err != nil {
		return "", err
	}
	if size == 0 {
		return filepath.Join(c.originalPath, "placeholder.png"), nil
	}
	path := filepath.Join(c.thumbPath, fmt.Sprintf("%d", size), "placeholder.jpg")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	orig, err := c.PlaceholderPath(0)
	if err != nil {
		return "", err
	}
	return c.thumbFromPlaceholder(orig, size)
}

func (c *Cache) thumbFromPlaceholder(original string, size int) (string, error) {
	if _, ok := SupportedThumbSizes[size]; !ok {
		return "", fmt.Errorf("unsupported thumb size %d", size)
	}
	if err := c.ensureLayout(); err != nil {
		return "", err
	}
	dir := filepath.Join(c.thumbPath, fmt.Sprintf("%d", size))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(dir, "placeholder.jpg")
	if _, err := os.Stat(target); err == nil {
		return target, nil
	}
	originalData, err := os.ReadFile(original)
	if err != nil {
		return "", err
	}
	img, _, err := image.Decode(bytes.NewReader(originalData))
	if err != nil {
		return "", err
	}
	thumb := resizeNearest(img, size, size)

	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, thumb, &jpeg.Options{Quality: 85}); err != nil {
		return "", err
	}
	if err := writeFileAtomically(target, encoded.Bytes(), 0o644); err != nil {
		return "", err
	}
	return target, nil
}

func (c *Cache) ensurePlaceholder() error {
	placeholder := filepath.Join(c.originalPath, "placeholder.png")
	if _, err := os.Stat(placeholder); err == nil {
		return nil
	}

	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			if (x+y)%2 == 0 {
				img.Set(x, y, color.RGBA{R: 42, G: 56, B: 84, A: 255})
			} else {
				img.Set(x, y, color.RGBA{R: 30, G: 42, B: 64, A: 255})
			}
		}
	}

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		return fmt.Errorf("encode placeholder: %w", err)
	}
	if err := writeFileAtomically(placeholder, encoded.Bytes(), 0o644); err != nil {
		return fmt.Errorf("create placeholder: %w", err)
	}
	return nil
}

func (c *Cache) ensureLayout() error {
	for _, p := range []string{c.rootPath, c.originalPath, c.thumbPath} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("create cover cache dir %s: %w", p, err)
		}
	}
	if err := c.ensurePlaceholder(); err != nil {
		return err
	}
	return nil
}

func extByMIME(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/png":
		return ".png"
	default:
		return ".jpg"
	}
}

func resizeNearest(src image.Image, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	sb := src.Bounds()
	sw := sb.Dx()
	sh := sb.Dy()
	if sw == 0 || sh == 0 {
		return dst
	}

	for y := 0; y < height; y++ {
		sy := sb.Min.Y + (y*sh)/height
		for x := 0; x < width; x++ {
			sx := sb.Min.X + (x*sw)/width
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
