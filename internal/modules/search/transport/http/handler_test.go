package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"music-server/internal/domain"
	searchservice "music-server/internal/modules/search/service"
)

type fakeSearchService struct{}

func (f *fakeSearchService) Search(context.Context, searchservice.Params) (searchservice.Result, error) {
	return searchservice.Result{
		Artists: []domain.Artist{{ID: "a1", Name: "Artist 1", CreatedAt: time.Now(), UpdatedAt: time.Now()}},
		Albums:  []domain.Album{{ID: "al1", Title: "Album 1", ArtistID: "a1", Year: 2020, CreatedAt: time.Now(), UpdatedAt: time.Now()}},
		Tracks:  []domain.Track{{ID: "t1", Title: "Track 1", AlbumID: "al1", ArtistID: "a1", TrackNumber: 1, Duration: time.Second, FilePath: "/music/t1.mp3", Codec: "mp3", Bitrate: 320, CreatedAt: time.Now(), UpdatedAt: time.Now()}},
	}, nil
}

func TestSearch(t *testing.T) {
	h := NewHandler(&fakeSearchService{})
	req := httptest.NewRequest(http.MethodGet, "/api/search?q=test", nil)
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"artists"`) || !strings.Contains(body, `"albums"`) || !strings.Contains(body, `"tracks"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}
