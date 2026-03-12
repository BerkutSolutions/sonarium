package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"music-server/internal/appmeta"
	authservice "music-server/internal/modules/auth/service"
	libraryrepo "music-server/internal/modules/library/repository"
	libraryservice "music-server/internal/modules/library/service"
	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Service interface {
	Home(ctx context.Context, userID string, limit int) (libraryrepo.HomeData, error)
	RandomAlbums(ctx context.Context, limit int) ([]libraryrepo.AlbumCard, error)
	ArtistAlbumCounts(ctx context.Context) ([]libraryrepo.ArtistAlbumCount, error)
	DeleteTrack(ctx context.Context, trackID string) error
	RenameTrack(ctx context.Context, trackID, title string) error
	DeleteAlbum(ctx context.Context, albumID string) error
	RenameAlbum(ctx context.Context, albumID, title string) error
	UpdateTrack(ctx context.Context, trackID string, input libraryrepo.TrackUpdateInput) error
	CreateAlbum(ctx context.Context, input libraryrepo.CreateAlbumInput) (string, error)
	UpdateAlbum(ctx context.Context, albumID string, input libraryrepo.AlbumUpdateInput) error
	MergeAlbum(ctx context.Context, albumID string, input libraryrepo.AlbumMergeInput) error
	UpdateArtist(ctx context.Context, artistID string, input libraryrepo.ArtistUpdateInput) error
	UpdateArtistCover(ctx context.Context, artistID string, data []byte, mimeType string) error
	DeleteArtist(ctx context.Context, artistID string) error
	ToggleFavoriteTrack(ctx context.Context, userID, trackID string) (bool, error)
	ToggleFavoriteAlbum(ctx context.Context, userID, albumID string) (bool, error)
	ToggleFavoriteArtist(ctx context.Context, userID, artistID string) (bool, error)
}

type Management interface {
	LibraryPath() string
	ScanStatus() libraryservice.ScanState
	TriggerScan() bool
	SaveUpload(ctx context.Context, fileName string, src io.Reader) (string, error)
	Settings(ctx context.Context) libraryservice.SettingsInfo
	CheckUpdates(ctx context.Context) (*appmeta.UpdateCheckResult, error)
}

type Handler struct {
	service    Service
	management Management
}

type renameRequest struct {
	Title string `json:"title"`
}

type updateTrackRequest struct {
	Title    string `json:"title"`
	AlbumID  string `json:"album_id"`
	ArtistID string `json:"artist_id"`
}

type createAlbumRequest struct {
	Title    string `json:"title"`
	ArtistID string `json:"artist_id"`
	Year     int    `json:"year"`
}

type updateAlbumRequest struct {
	Title    string `json:"title"`
	ArtistID string `json:"artist_id"`
	Year     int    `json:"year"`
}

type mergeAlbumRequest struct {
	TargetAlbumID string `json:"target_album_id"`
}

type updateArtistRequest struct {
	Name             string `json:"name"`
	ExistingArtistID string `json:"existing_artist_id"`
}

func NewHandler(service Service, management Management) *Handler {
	return &Handler{service: service, management: management}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 12)
	data, err := h.service.Home(r.Context(), currentUserID(r.Context()), limit)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to build home dashboard"))
		return
	}
	response.WriteOK(w, map[string]any{"data": data})
}

func (h *Handler) RandomAlbums(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 12)
	albums, err := h.service.RandomAlbums(r.Context(), limit)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to load random albums"))
		return
	}
	response.WriteOK(w, map[string]any{"data": albums})
}

func (h *Handler) ArtistAlbumCounts(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ArtistAlbumCounts(r.Context())
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to load artist stats"))
		return
	}
	response.WriteOK(w, map[string]any{"data": items})
}

func (h *Handler) ToggleFavoriteTrack(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "track_id")
	if trackID == "" {
		response.WriteError(w, apperrors.NewBadRequest("track id is required"))
		return
	}
	active, err := h.service.ToggleFavoriteTrack(r.Context(), currentUserID(r.Context()), trackID)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to toggle track favorite"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"favorite": active}})
}

func (h *Handler) ToggleFavoriteAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "album_id")
	if albumID == "" {
		response.WriteError(w, apperrors.NewBadRequest("album id is required"))
		return
	}
	active, err := h.service.ToggleFavoriteAlbum(r.Context(), currentUserID(r.Context()), albumID)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to toggle album favorite"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"favorite": active}})
}

func (h *Handler) ToggleFavoriteArtist(w http.ResponseWriter, r *http.Request) {
	artistID := chi.URLParam(r, "artist_id")
	if artistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("artist id is required"))
		return
	}
	active, err := h.service.ToggleFavoriteArtist(r.Context(), currentUserID(r.Context()), artistID)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to toggle artist favorite"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"favorite": active}})
}

func (h *Handler) Scan(w http.ResponseWriter, r *http.Request) {
	if started := h.management.TriggerScan(); !started {
		response.WriteOK(w, map[string]any{"data": map[string]any{
			"status":  "already_scanning",
			"message": "scan already in progress",
		}})
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]any{
		"status": "started",
	}})
}

func (h *Handler) ScanStatus(w http.ResponseWriter, r *http.Request) {
	state := h.management.ScanStatus()
	response.WriteOK(w, map[string]any{
		"data": map[string]any{
			"library_path": h.management.LibraryPath(),
			"scan":         state,
		},
	})
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid multipart form"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		response.WriteError(w, apperrors.NewBadRequest("file is required"))
		return
	}
	defer file.Close()

	path, err := h.management.SaveUpload(r.Context(), header.Filename, file)
	if err != nil {
		response.WriteError(w, apperrors.NewBadRequest(err.Error()))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]any{
		"stored_path": path,
		"status":      "uploaded",
	}})
}

func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	settings := h.management.Settings(r.Context())
	current := authservice.CurrentUser(r.Context())
	if current != nil && current.Role == authservice.RoleAdmin {
		settings.CanCheckUpdates = true
	} else {
		settings.UpdateCheck = nil
	}
	response.WriteOK(w, map[string]any{"data": settings})
}

func (h *Handler) CheckUpdates(w http.ResponseWriter, r *http.Request) {
	current := authservice.CurrentUser(r.Context())
	if current == nil {
		response.WriteError(w, apperrors.AppError{Code: "unauthorized", Message: "login required", HTTPStatus: http.StatusUnauthorized})
		return
	}
	if current.Role != authservice.RoleAdmin {
		response.WriteError(w, apperrors.AppError{Code: "forbidden", Message: "forbidden", HTTPStatus: http.StatusForbidden})
		return
	}
	result, err := h.management.CheckUpdates(r.Context())
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to check updates"))
		return
	}
	response.WriteOK(w, map[string]any{"data": result})
}

func (h *Handler) DeleteTrack(w http.ResponseWriter, r *http.Request) {
	trackID := strings.TrimSpace(chi.URLParam(r, "track_id"))
	if trackID == "" {
		response.WriteError(w, apperrors.NewBadRequest("track id is required"))
		return
	}
	if err := h.service.DeleteTrack(r.Context(), trackID); err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to delete track"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) RenameTrack(w http.ResponseWriter, r *http.Request) {
	trackID := strings.TrimSpace(chi.URLParam(r, "track_id"))
	if trackID == "" {
		response.WriteError(w, apperrors.NewBadRequest("track id is required"))
		return
	}
	var req renameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		response.WriteError(w, apperrors.NewBadRequest("title is required"))
		return
	}
	if err := h.service.RenameTrack(r.Context(), trackID, req.Title); err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to rename track"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := strings.TrimSpace(chi.URLParam(r, "album_id"))
	if albumID == "" {
		response.WriteError(w, apperrors.NewBadRequest("album id is required"))
		return
	}
	if err := h.service.DeleteAlbum(r.Context(), albumID); err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to delete album"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) RenameAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := strings.TrimSpace(chi.URLParam(r, "album_id"))
	if albumID == "" {
		response.WriteError(w, apperrors.NewBadRequest("album id is required"))
		return
	}
	var req renameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		response.WriteError(w, apperrors.NewBadRequest("title is required"))
		return
	}
	if err := h.service.RenameAlbum(r.Context(), albumID, req.Title); err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to rename album"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) UpdateTrack(w http.ResponseWriter, r *http.Request) {
	trackID := strings.TrimSpace(chi.URLParam(r, "track_id"))
	if trackID == "" {
		response.WriteError(w, apperrors.NewBadRequest("track id is required"))
		return
	}
	var req updateTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.AlbumID = strings.TrimSpace(req.AlbumID)
	req.ArtistID = strings.TrimSpace(req.ArtistID)
	if req.Title == "" {
		response.WriteError(w, apperrors.NewBadRequest("title is required"))
		return
	}
	if req.AlbumID == "" {
		response.WriteError(w, apperrors.NewBadRequest("album id is required"))
		return
	}
	if req.ArtistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("artist id is required"))
		return
	}
	if err := h.service.UpdateTrack(r.Context(), trackID, libraryrepo.TrackUpdateInput{
		Title:    req.Title,
		AlbumID:  req.AlbumID,
		ArtistID: req.ArtistID,
	}); err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to update track"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	var req createAlbumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.ArtistID = strings.TrimSpace(req.ArtistID)
	if req.Title == "" {
		response.WriteError(w, apperrors.NewBadRequest("title is required"))
		return
	}
	if req.ArtistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("artist id is required"))
		return
	}
	id, err := h.service.CreateAlbum(r.Context(), libraryrepo.CreateAlbumInput{
		Title:    req.Title,
		ArtistID: req.ArtistID,
		Year:     req.Year,
	})
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to create album"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]string{"id": id}})
}

func (h *Handler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := strings.TrimSpace(chi.URLParam(r, "album_id"))
	if albumID == "" {
		response.WriteError(w, apperrors.NewBadRequest("album id is required"))
		return
	}
	var req updateAlbumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.ArtistID = strings.TrimSpace(req.ArtistID)
	if req.Title == "" {
		response.WriteError(w, apperrors.NewBadRequest("title is required"))
		return
	}
	if req.ArtistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("artist id is required"))
		return
	}
	if err := h.service.UpdateAlbum(r.Context(), albumID, libraryrepo.AlbumUpdateInput{
		Title:    req.Title,
		ArtistID: req.ArtistID,
		Year:     req.Year,
	}); err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to update album"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) MergeAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := strings.TrimSpace(chi.URLParam(r, "album_id"))
	if albumID == "" {
		response.WriteError(w, apperrors.NewBadRequest("album id is required"))
		return
	}
	var req mergeAlbumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	req.TargetAlbumID = strings.TrimSpace(req.TargetAlbumID)
	if req.TargetAlbumID == "" {
		response.WriteError(w, apperrors.NewBadRequest("target album id is required"))
		return
	}
	if req.TargetAlbumID == albumID {
		response.WriteError(w, apperrors.NewBadRequest("target album must be different"))
		return
	}
	if err := h.service.MergeAlbum(r.Context(), albumID, libraryrepo.AlbumMergeInput{
		TargetAlbumID: req.TargetAlbumID,
	}); err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to merge album"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) UpdateArtist(w http.ResponseWriter, r *http.Request) {
	artistID := strings.TrimSpace(chi.URLParam(r, "artist_id"))
	if artistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("artist id is required"))
		return
	}
	var req updateArtistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.ExistingArtistID = strings.TrimSpace(req.ExistingArtistID)
	if req.Name == "" {
		response.WriteError(w, apperrors.NewBadRequest("name is required"))
		return
	}
	if err := h.service.UpdateArtist(r.Context(), artistID, libraryrepo.ArtistUpdateInput{
		Name:              req.Name,
		MergeIntoArtistID: req.ExistingArtistID,
	}); err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to update artist"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) DeleteArtist(w http.ResponseWriter, r *http.Request) {
	artistID := strings.TrimSpace(chi.URLParam(r, "artist_id"))
	if artistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("artist id is required"))
		return
	}
	if err := h.service.DeleteArtist(r.Context(), artistID); err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to delete artist"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) UpdateArtistCover(w http.ResponseWriter, r *http.Request) {
	artistID := strings.TrimSpace(chi.URLParam(r, "artist_id"))
	if artistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("artist id is required"))
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid multipart form"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		response.WriteError(w, apperrors.NewBadRequest("file is required"))
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to read file"))
		return
	}
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	if err := h.service.UpdateArtistCover(r.Context(), artistID, data, mimeType); err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to update artist cover"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func parseLimit(r *http.Request, fallback int) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	if value > 50 {
		return 50
	}
	return value
}

func currentUserID(ctx context.Context) string {
	if userID := authservice.CurrentUserID(ctx); userID != "" {
		return userID
	}
	return libraryservice.DefaultUserID
}
