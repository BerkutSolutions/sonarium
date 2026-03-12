package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"music-server/internal/domain"
	tracksservice "music-server/internal/modules/tracks/service"
)

type fakeTrackService struct{}

func (f *fakeTrackService) List(context.Context, tracksservice.ListParams) ([]domain.Track, error) {
	return []domain.Track{{ID: "t1", Title: "Track 1", AlbumID: "al1", ArtistID: "a1", TrackNumber: 1, Duration: time.Second, FilePath: "/music/t1.mp3", Codec: "mp3", Bitrate: 320, CreatedAt: time.Now(), UpdatedAt: time.Now()}}, nil
}
func (f *fakeTrackService) GetByID(context.Context, string) (domain.Track, error) {
	return domain.Track{ID: "t1", Title: "Track 1", AlbumID: "al1", ArtistID: "a1", TrackNumber: 1, Duration: time.Second, FilePath: "/music/t1.mp3", Codec: "mp3", Bitrate: 320, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}
func (f *fakeTrackService) ListByAlbumID(context.Context, string, tracksservice.ListParams) ([]domain.Track, error) {
	return []domain.Track{{ID: "t1", Title: "Track 1", AlbumID: "al1", ArtistID: "a1", TrackNumber: 1, Duration: time.Second, FilePath: "/music/t1.mp3", Codec: "mp3", Bitrate: 320, CreatedAt: time.Now(), UpdatedAt: time.Now()}}, nil
}

func TestListTracks(t *testing.T) {
	h := NewHandler(&fakeTrackService{})
	r := chi.NewRouter()
	r.Get("/api/tracks", h.ListTracks)

	req := httptest.NewRequest(http.MethodGet, "/api/tracks?sort=name", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"data"`) {
		t.Fatalf("expected data envelope, got %s", rec.Body.String())
	}
}
