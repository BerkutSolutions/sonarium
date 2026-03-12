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
	albumsservice "music-server/internal/modules/albums/service"
)

type fakeAlbumService struct{}

func (f *fakeAlbumService) List(context.Context, albumsservice.ListParams) ([]domain.Album, error) {
	return []domain.Album{{ID: "al1", Title: "Album 1", ArtistID: "a1", Year: 2020, CreatedAt: time.Now(), UpdatedAt: time.Now()}}, nil
}
func (f *fakeAlbumService) GetByID(context.Context, string) (domain.Album, error) {
	return domain.Album{ID: "al1", Title: "Album 1", ArtistID: "a1", Year: 2020, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}
func (f *fakeAlbumService) ListByArtistID(context.Context, string, albumsservice.ListParams) ([]domain.Album, error) {
	return []domain.Album{{ID: "al1", Title: "Album 1", ArtistID: "a1", Year: 2020, CreatedAt: time.Now(), UpdatedAt: time.Now()}}, nil
}

func TestListAlbums(t *testing.T) {
	h := NewHandler(&fakeAlbumService{})
	r := chi.NewRouter()
	r.Get("/api/albums", h.ListAlbums)

	req := httptest.NewRequest(http.MethodGet, "/api/albums?sort=year", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"data"`) {
		t.Fatalf("expected data envelope, got %s", rec.Body.String())
	}
}
