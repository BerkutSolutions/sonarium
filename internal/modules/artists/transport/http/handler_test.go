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
	artistsservice "music-server/internal/modules/artists/service"
)

type fakeArtistService struct{}

func (f *fakeArtistService) List(context.Context, artistsservice.ListParams) ([]domain.Artist, error) {
	return []domain.Artist{{ID: "a1", Name: "Artist 1", CreatedAt: time.Now(), UpdatedAt: time.Now()}}, nil
}
func (f *fakeArtistService) GetByID(context.Context, string) (domain.Artist, error) {
	return domain.Artist{ID: "a1", Name: "Artist 1", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func TestListArtists(t *testing.T) {
	h := NewHandler(&fakeArtistService{})
	r := chi.NewRouter()
	r.Get("/api/artists", h.ListArtists)

	req := httptest.NewRequest(http.MethodGet, "/api/artists?limit=10&offset=0&sort=name", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"data"`) {
		t.Fatalf("expected data envelope, got %s", rec.Body.String())
	}
}
