package http

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"music-server/internal/domain"
	streamservice "music-server/internal/modules/stream/service"
)

type fakeTrackRepository struct {
	track domain.Track
	err   error
}

func (f *fakeTrackRepository) List(context.Context) ([]domain.Track, error) {
	return nil, nil
}
func (f *fakeTrackRepository) GetByID(context.Context, string) (domain.Track, error) {
	if f.err != nil {
		return domain.Track{}, f.err
	}
	return f.track, nil
}
func (f *fakeTrackRepository) GetByFilePath(context.Context, string) (domain.Track, error) {
	return domain.Track{}, sql.ErrNoRows
}
func (f *fakeTrackRepository) ListByAlbumID(context.Context, string) ([]domain.Track, error) {
	return nil, nil
}
func (f *fakeTrackRepository) ListByArtistID(context.Context, string) ([]domain.Track, error) {
	return nil, nil
}
func (f *fakeTrackRepository) Search(context.Context, string) ([]domain.Track, error) {
	return nil, nil
}
func (f *fakeTrackRepository) Upsert(context.Context, domain.Track) error {
	return nil
}

func TestStreamTrack_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "song.mp3")
	if err := os.WriteFile(filePath, []byte("abcdefghi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	service := streamservice.New(&fakeTrackRepository{
		track: domain.Track{
			ID:        "11111111-1111-1111-1111-111111111111",
			FilePath:  filePath,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	})
	handler := NewHandler(zap.NewNop(), service, nil)

	router := chi.NewRouter()
	router.Get("/api/stream/{track_id}", handler.StreamTrack)

	req := httptest.NewRequest(http.MethodGet, "/api/stream/11111111-1111-1111-1111-111111111111", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "audio/mpeg" {
		t.Fatalf("expected content-type audio/mpeg, got %q", got)
	}
	if body := rec.Body.String(); body != "abcdefghi" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestStreamTrack_RangeRequest(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "song.ogg")
	if err := os.WriteFile(filePath, []byte("0123456789"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	service := streamservice.New(&fakeTrackRepository{
		track: domain.Track{
			ID:        "22222222-2222-2222-2222-222222222222",
			FilePath:  filePath,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	})
	handler := NewHandler(zap.NewNop(), service, nil)

	router := chi.NewRouter()
	router.Get("/api/stream/{track_id}", handler.StreamTrack)

	req := httptest.NewRequest(http.MethodGet, "/api/stream/22222222-2222-2222-2222-222222222222", nil)
	req.Header.Set("Range", "bytes=0-3")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("expected status 206, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "0123" {
		t.Fatalf("unexpected partial body: %q", body)
	}
}

func TestStreamTrack_NotFound(t *testing.T) {
	t.Parallel()

	service := streamservice.New(&fakeTrackRepository{err: sql.ErrNoRows})
	handler := NewHandler(zap.NewNop(), service, nil)

	router := chi.NewRouter()
	router.Get("/api/stream/{track_id}", handler.StreamTrack)

	req := httptest.NewRequest(http.MethodGet, "/api/stream/11111111-1111-1111-1111-111111111111", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	expected := `{"error":{"code":"not_found","message":"track not found"}}`
	if body != expected {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestStreamTrack_InvalidTrackID(t *testing.T) {
	t.Parallel()

	service := streamservice.New(&fakeTrackRepository{})
	handler := NewHandler(zap.NewNop(), service, nil)

	router := chi.NewRouter()
	router.Get("/api/stream/{track_id}", handler.StreamTrack)

	req := httptest.NewRequest(http.MethodGet, "/api/stream/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	expected := `{"error":{"code":"invalid_request","message":"invalid track id"}}`
	if body != expected {
		t.Fatalf("unexpected body: %s", body)
	}
}
