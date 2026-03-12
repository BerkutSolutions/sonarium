package service

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type Request struct {
	InputPath string
	Format    string
	Bitrate   int
}

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) ShouldTranscode(format string, bitrate int) bool {
	return strings.TrimSpace(format) != "" || bitrate > 0
}

func (s *Service) OpenReader(ctx context.Context, req Request) (io.ReadCloser, string, error) {
	targetFormat, codec, mime, err := resolveTarget(req.Format)
	if err != nil {
		return nil, "", err
	}
	bitrate := req.Bitrate
	if bitrate <= 0 {
		bitrate = 192
	}
	args := []string{
		"-i", req.InputPath,
		"-vn",
		"-ac", "2",
		"-b:a", strconv.Itoa(bitrate) + "k",
		"-f", targetFormat,
	}
	if codec != "" {
		args = append(args, "-c:a", codec)
	}
	args = append(args, "pipe:1")

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stderr, _ := cmd.StderrPipe()
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", fmt.Errorf("ffmpeg stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, "", fmt.Errorf("ffmpeg start: %w", err)
	}
	return &cmdReadCloser{
		reader: pipe,
		cmd:    cmd,
		stderr: stderr,
	}, mime, nil
}

func resolveTarget(format string) (container string, codec string, mime string, err error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "mp3":
		return "mp3", "libmp3lame", "audio/mpeg", nil
	case "opus":
		return "opus", "libopus", "audio/ogg", nil
	case "aac":
		return "adts", "aac", "audio/aac", nil
	default:
		return "", "", "", fmt.Errorf("unsupported transcode format")
	}
}

type cmdReadCloser struct {
	reader io.ReadCloser
	cmd    *exec.Cmd
	stderr io.ReadCloser
}

func (c *cmdReadCloser) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *cmdReadCloser) Close() error {
	_ = c.reader.Close()
	if c.stderr != nil {
		_ = c.stderr.Close()
	}
	return c.cmd.Wait()
}
