package http

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"music-server/internal/domain"
	searchservice "music-server/internal/modules/search/service"
	streamservice "music-server/internal/modules/stream/service"
	"music-server/internal/modules/subsonic/mapper"
	subsonicservice "music-server/internal/modules/subsonic/service"
)

type Service interface {
	ListArtists(ctx context.Context) ([]domain.Artist, error)
	GetArtist(ctx context.Context, id string) (domain.Artist, []domain.Album, error)
	GetAlbum(ctx context.Context, id string) (domain.Album, []domain.Track, error)
	GetAlbumList(ctx context.Context, size, offset int) ([]domain.Album, error)
	GetSong(ctx context.Context, id string) (domain.Track, error)
	GetPlaylists(ctx context.Context) ([]domain.Playlist, error)
	GetPlaylist(ctx context.Context, id string) (domain.Playlist, []domain.Track, error)
	Search3(ctx context.Context, query string, limit, offset int) (searchservice.Result, error)
	ResolveCoverArt(ctx context.Context, id string) (string, string, error)
	ResolveStream(ctx context.Context, trackID string) (streamservice.StreamableTrack, error)
}

type Handler struct {
	authenticator *subsonicservice.Authenticator
	service       Service
}

func NewHandler(authenticator *subsonicservice.Authenticator, service Service) *Handler {
	return &Handler{
		authenticator: authenticator,
		service:       service,
	}
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/ping.view", h.Ping)
	r.Get("/getLicense.view", h.GetLicense)
	r.Get("/getArtists.view", h.GetArtists)
	r.Get("/getArtist.view", h.GetArtist)
	r.Get("/getAlbum.view", h.GetAlbum)
	r.Get("/getAlbumList.view", h.GetAlbumList)
	r.Get("/getSong.view", h.GetSong)
	r.Get("/getPlaylists.view", h.GetPlaylists)
	r.Get("/getPlaylist.view", h.GetPlaylist)
	r.Get("/search3.view", h.Search3)
	r.Get("/getCoverArt.view", h.GetCoverArt)
	r.Get("/stream.view", h.Stream)
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	format, ok := h.authorize(w, r)
	if !ok {
		return
	}
	mapper.WriteResponse(w, format, mapper.NewSuccess())
}

func (h *Handler) GetLicense(w http.ResponseWriter, r *http.Request) {
	format, ok := h.authorize(w, r)
	if !ok {
		return
	}
	resp := mapper.NewSuccess()
	resp.License = &mapper.License{Valid: true}
	mapper.WriteResponse(w, format, resp)
}

func (h *Handler) GetArtists(w http.ResponseWriter, r *http.Request) {
	format, ok := h.authorize(w, r)
	if !ok {
		return
	}
	artists, err := h.service.ListArtists(r.Context())
	if err != nil {
		h.writeSubsonicError(w, format, err)
		return
	}
	resp := mapper.NewSuccess()
	resp.Artists = mapper.Artists(artists)
	mapper.WriteResponse(w, format, resp)
}

func (h *Handler) GetArtist(w http.ResponseWriter, r *http.Request) {
	format, ok := h.authorize(w, r)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if strings.TrimSpace(id) == "" {
		h.writeSubsonicError(w, format, subsonicservice.ErrMissingProtocolParams)
		return
	}
	artist, albums, err := h.service.GetArtist(r.Context(), id)
	if err != nil {
		h.writeSubsonicError(w, format, err)
		return
	}
	resp := mapper.NewSuccess()
	resp.Artist = mapper.ArtistWithAlbums(artist, albums)
	mapper.WriteResponse(w, format, resp)
}

func (h *Handler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	format, ok := h.authorize(w, r)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if strings.TrimSpace(id) == "" {
		h.writeSubsonicError(w, format, subsonicservice.ErrMissingProtocolParams)
		return
	}
	album, songs, err := h.service.GetAlbum(r.Context(), id)
	if err != nil {
		h.writeSubsonicError(w, format, err)
		return
	}
	resp := mapper.NewSuccess()
	resp.Album = mapper.AlbumWithSongs(album, songs)
	mapper.WriteResponse(w, format, resp)
}

func (h *Handler) GetAlbumList(w http.ResponseWriter, r *http.Request) {
	format, ok := h.authorize(w, r)
	if !ok {
		return
	}
	size := parseInt(r.URL.Query().Get("size"), 50)
	offset := parseInt(r.URL.Query().Get("offset"), 0)
	albums, err := h.service.GetAlbumList(r.Context(), size, offset)
	if err != nil {
		h.writeSubsonicError(w, format, err)
		return
	}
	resp := mapper.NewSuccess()
	resp.AlbumList = mapper.AlbumListFromDomain(albums)
	mapper.WriteResponse(w, format, resp)
}

func (h *Handler) GetSong(w http.ResponseWriter, r *http.Request) {
	format, ok := h.authorize(w, r)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if strings.TrimSpace(id) == "" {
		h.writeSubsonicError(w, format, subsonicservice.ErrMissingProtocolParams)
		return
	}
	song, err := h.service.GetSong(r.Context(), id)
	if err != nil {
		h.writeSubsonicError(w, format, err)
		return
	}
	resp := mapper.NewSuccess()
	mapped := mapper.SongFromTrack(song)
	resp.Song = &mapped
	mapper.WriteResponse(w, format, resp)
}

func (h *Handler) GetPlaylists(w http.ResponseWriter, r *http.Request) {
	format, ok := h.authorize(w, r)
	if !ok {
		return
	}
	playlists, err := h.service.GetPlaylists(r.Context())
	if err != nil {
		h.writeSubsonicError(w, format, err)
		return
	}
	resp := mapper.NewSuccess()
	resp.Playlists = mapper.PlaylistsFromDomain(playlists)
	mapper.WriteResponse(w, format, resp)
}

func (h *Handler) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	format, ok := h.authorize(w, r)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if strings.TrimSpace(id) == "" {
		h.writeSubsonicError(w, format, subsonicservice.ErrMissingProtocolParams)
		return
	}
	playlist, tracks, err := h.service.GetPlaylist(r.Context(), id)
	if err != nil {
		h.writeSubsonicError(w, format, err)
		return
	}
	resp := mapper.NewSuccess()
	resp.Playlist = mapper.PlaylistWithSongs(playlist, tracks)
	mapper.WriteResponse(w, format, resp)
}

func (h *Handler) Search3(w http.ResponseWriter, r *http.Request) {
	format, ok := h.authorize(w, r)
	if !ok {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("q"))
	}
	if query == "" {
		h.writeSubsonicError(w, format, subsonicservice.ErrMissingProtocolParams)
		return
	}
	limit := parseInt(r.URL.Query().Get("count"), 20)
	offset := parseInt(r.URL.Query().Get("offset"), 0)
	result, err := h.service.Search3(r.Context(), query, limit, offset)
	if err != nil {
		h.writeSubsonicError(w, format, err)
		return
	}
	resp := mapper.NewSuccess()
	resp.SearchResult3 = mapper.SearchResultFromDomain(result)
	mapper.WriteResponse(w, format, resp)
}

func (h *Handler) GetCoverArt(w http.ResponseWriter, r *http.Request) {
	_, ok := h.authorize(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		h.writeSubsonicError(w, "json", subsonicservice.ErrMissingProtocolParams)
		return
	}
	path, mimeType, err := h.service.ResolveCoverArt(r.Context(), id)
	if err != nil {
		format := r.URL.Query().Get("f")
		h.writeSubsonicError(w, format, err)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		format := r.URL.Query().Get("f")
		h.writeSubsonicError(w, format, err)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		format := r.URL.Query().Get("f")
		h.writeSubsonicError(w, format, err)
		return
	}

	w.Header().Set("Content-Type", mimeType)
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
}

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	_, ok := h.authorize(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		format := r.URL.Query().Get("f")
		h.writeSubsonicError(w, format, subsonicservice.ErrMissingProtocolParams)
		return
	}
	resolved, err := h.service.ResolveStream(r.Context(), id)
	if err != nil {
		format := r.URL.Query().Get("f")
		h.writeSubsonicError(w, format, err)
		return
	}
	file, err := os.Open(resolved.FilePath)
	if err != nil {
		format := r.URL.Query().Get("f")
		h.writeSubsonicError(w, format, err)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", resolved.MIMEType)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, resolved.FileName, resolved.ModTime, file)
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) (string, bool) {
	params := subsonicservice.ProtocolParams{
		Username: r.URL.Query().Get("u"),
		Token:    r.URL.Query().Get("t"),
		Salt:     r.URL.Query().Get("s"),
		Version:  r.URL.Query().Get("v"),
		Client:   r.URL.Query().Get("c"),
		Format:   r.URL.Query().Get("f"),
	}
	if params.Format == "" {
		params.Format = "json"
	}
	if err := h.authenticator.Validate(params); err != nil {
		h.writeSubsonicError(w, params.Format, err)
		return params.Format, false
	}
	return params.Format, true
}

func (h *Handler) writeSubsonicError(w http.ResponseWriter, format string, err error) {
	code, msg := mapper.ErrorFrom(err)
	resp := mapper.WithError(mapper.NewSuccess(), code, msg)
	mapper.WriteResponse(w, format, resp)
}

func parseInt(value string, fallback int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if n < 0 {
		return fallback
	}
	return n
}
