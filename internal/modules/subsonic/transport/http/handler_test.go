package http

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"music-server/internal/domain"
	searchservice "music-server/internal/modules/search/service"
	streamservice "music-server/internal/modules/stream/service"
	subsonicservice "music-server/internal/modules/subsonic/service"
)

type fakeSubsonicService struct {
	streamFile string
}

func (f *fakeSubsonicService) ListArtists(context.Context) ([]domain.Artist, error) {
	return []domain.Artist{{ID: "a1", Name: "Artist 1"}}, nil
}
func (f *fakeSubsonicService) GetArtist(context.Context, string) (domain.Artist, []domain.Album, error) {
	return domain.Artist{ID: "a1", Name: "Artist 1"}, []domain.Album{{ID: "al1", Title: "Album 1", ArtistID: "a1"}}, nil
}
func (f *fakeSubsonicService) GetAlbum(context.Context, string) (domain.Album, []domain.Track, error) {
	return domain.Album{ID: "al1", Title: "Album 1", ArtistID: "a1"}, []domain.Track{{ID: "t1", Title: "Track 1", AlbumID: "al1", ArtistID: "a1", Duration: time.Second, TrackNumber: 1, Bitrate: 320, FilePath: "/music/t1.mp3"}}, nil
}
func (f *fakeSubsonicService) GetAlbumList(context.Context, int, int) ([]domain.Album, error) {
	return []domain.Album{{ID: "al1", Title: "Album 1", ArtistID: "a1"}}, nil
}
func (f *fakeSubsonicService) GetSong(context.Context, string) (domain.Track, error) {
	return domain.Track{ID: "t1", Title: "Track 1", AlbumID: "al1", ArtistID: "a1", Duration: time.Second, TrackNumber: 1, Bitrate: 320, FilePath: "/music/t1.mp3"}, nil
}
func (f *fakeSubsonicService) GetPlaylists(context.Context) ([]domain.Playlist, error) {
	return []domain.Playlist{{ID: "p1", Name: "Playlist 1"}}, nil
}
func (f *fakeSubsonicService) GetPlaylist(context.Context, string) (domain.Playlist, []domain.Track, error) {
	return domain.Playlist{ID: "p1", Name: "Playlist 1"}, []domain.Track{{ID: "t1", Title: "Track 1", AlbumID: "al1", ArtistID: "a1", Duration: time.Second, TrackNumber: 1, Bitrate: 320, FilePath: "/music/t1.mp3"}}, nil
}
func (f *fakeSubsonicService) Search3(context.Context, string, int, int) (searchservice.Result, error) {
	return searchservice.Result{
		Artists: []domain.Artist{{ID: "a1", Name: "Artist 1"}},
		Albums:  []domain.Album{{ID: "al1", Title: "Album 1", ArtistID: "a1"}},
		Tracks:  []domain.Track{{ID: "t1", Title: "Track 1", AlbumID: "al1", ArtistID: "a1", Duration: time.Second, TrackNumber: 1, Bitrate: 320, FilePath: "/music/t1.mp3"}},
	}, nil
}
func (f *fakeSubsonicService) ResolveCoverArt(context.Context, string) (string, string, error) {
	return "", "", subsonicservice.ErrCoverArtNotFound
}
func (f *fakeSubsonicService) ResolveStream(context.Context, string) (streamservice.StreamableTrack, error) {
	return streamservice.StreamableTrack{
		TrackID:  "t1",
		FilePath: f.streamFile,
		ModTime:  time.Now().UTC(),
		MIMEType: "audio/mpeg",
		FileName: filepath.Base(f.streamFile),
	}, nil
}

func TestPing(t *testing.T) {
	handler := newTestHandler(t, "")
	req := httptest.NewRequest(http.MethodGet, "/rest/ping.view?"+authQuery("json"), nil)
	rec := httptest.NewRecorder()
	handler.Ping(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestGetArtists(t *testing.T) {
	handler := newTestHandler(t, "")
	req := httptest.NewRequest(http.MethodGet, "/rest/getArtists.view?"+authQuery("json"), nil)
	rec := httptest.NewRecorder()
	handler.GetArtists(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"artists"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestGetAlbum(t *testing.T) {
	handler := newTestHandler(t, "")
	req := httptest.NewRequest(http.MethodGet, "/rest/getAlbum.view?id=al1&"+authQuery("json"), nil)
	rec := httptest.NewRecorder()
	handler.GetAlbum(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"album"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestSearch3(t *testing.T) {
	handler := newTestHandler(t, "")
	req := httptest.NewRequest(http.MethodGet, "/rest/search3.view?query=test&"+authQuery("json"), nil)
	rec := httptest.NewRecorder()
	handler.Search3(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"searchResult3"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestStream(t *testing.T) {
	tmpDir := t.TempDir()
	audio := filepath.Join(tmpDir, "track.mp3")
	if err := os.WriteFile(audio, []byte("abcdefghij"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	handler := newTestHandler(t, audio)
	router := chi.NewRouter()
	router.Get("/rest/stream.view", handler.Stream)

	req := httptest.NewRequest(http.MethodGet, "/rest/stream.view?id=t1&"+authQuery("json"), nil)
	req.Header.Set("Range", "bytes=0-3")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("expected 206, got %d", rec.Code)
	}
	if rec.Body.String() != "abcd" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestAuthValidationFailure(t *testing.T) {
	handler := newTestHandler(t, "")
	req := httptest.NewRequest(http.MethodGet, "/rest/ping.view?u=admin&t=wrong&s=salt&v=1.16.1&c=test&f=json", nil)
	rec := httptest.NewRecorder()
	handler.Ping(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with subsonic failed status, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"failed"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func newTestHandler(t *testing.T, streamFile string) *Handler {
	t.Helper()
	return NewHandler(
		subsonicservice.NewAuthenticator(subsonicservice.AuthConfig{
			Username:   "admin",
			Password:   "secret",
			MinVersion: "1.16.1",
		}),
		&fakeSubsonicService{streamFile: streamFile},
	)
}

func authQuery(format string) string {
	salt := "salt"
	hash := md5.Sum([]byte("secret" + salt))
	token := hex.EncodeToString(hash[:])
	return "u=admin&t=" + token + "&s=" + salt + "&v=1.16.1&c=test-client&f=" + format
}
