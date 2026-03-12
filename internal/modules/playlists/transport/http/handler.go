package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"music-server/internal/domain"
	authservice "music-server/internal/modules/auth/service"
	playlistsservice "music-server/internal/modules/playlists/service"
	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Service interface {
	List(ctx context.Context, current *authservice.User, params playlistsservice.ListParams) ([]domain.Playlist, error)
	GetByID(ctx context.Context, current *authservice.User, id, shareToken string) (domain.Playlist, error)
	ListTracks(ctx context.Context, current *authservice.User, playlistID, shareToken string) ([]domain.Track, error)
	Create(ctx context.Context, current *authservice.User, name string) (domain.Playlist, error)
	AddTrack(ctx context.Context, current *authservice.User, playlistID string, trackID string, position int) error
	RemoveTrack(ctx context.Context, current *authservice.User, playlistID string, trackID string) error
	Delete(ctx context.Context, current *authservice.User, id string) error
	Rename(ctx context.Context, current *authservice.User, id, name string) error
	Update(ctx context.Context, current *authservice.User, id, name, description string) error
}

type Handler struct {
	service Service
}

type createPlaylistRequest struct {
	Name string `json:"name"`
}

type addTrackRequest struct {
	TrackID  string `json:"track_id"`
	Position int    `json:"position"`
}

type renamePlaylistRequest struct {
	Name string `json:"name"`
}

type updatePlaylistRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListPlaylists(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r)
	if err != nil {
		response.WriteError(w, apperrors.NewBadRequest(err.Error()))
		return
	}
	playlists, err := h.service.List(r.Context(), authservice.CurrentUser(r.Context()), params)
	if err != nil {
		writePlaylistError(w, err, "failed to list playlists")
		return
	}
	response.WriteOK(w, map[string]any{"data": playlists})
}

func (h *Handler) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	var req createPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		response.WriteError(w, apperrors.NewBadRequest("name is required"))
		return
	}

	playlist, err := h.service.Create(r.Context(), authservice.CurrentUser(r.Context()), req.Name)
	if err != nil {
		writePlaylistError(w, err, "failed to create playlist")
		return
	}
	response.WriteJSON(w, http.StatusCreated, map[string]any{"data": playlist})
}

func (h *Handler) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	playlistID := chi.URLParam(r, "id")
	if playlistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("playlist id is required"))
		return
	}

	shareToken := strings.TrimSpace(r.URL.Query().Get("share"))
	currentUser := authservice.CurrentUser(r.Context())
	playlist, err := h.service.GetByID(r.Context(), currentUser, playlistID, shareToken)
	if err != nil {
		writePlaylistError(w, err, "failed to load playlist")
		return
	}
	tracks, err := h.service.ListTracks(r.Context(), currentUser, playlistID, shareToken)
	if err != nil {
		writePlaylistError(w, err, "failed to load playlist tracks")
		return
	}

	response.WriteOK(w, map[string]any{
		"data": map[string]any{
			"playlist": playlist,
			"tracks":   tracks,
			"permissions": map[string]any{
				"can_edit":  playlist.AccessRole == "owner" || playlist.AccessRole == "editor",
				"can_share": playlist.AccessRole == "owner",
				"is_owner":  playlist.AccessRole == "owner",
				"role":      playlist.AccessRole,
			},
		},
	})
}

func (h *Handler) AddTrack(w http.ResponseWriter, r *http.Request) {
	playlistID := chi.URLParam(r, "id")
	if playlistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("playlist id is required"))
		return
	}

	var req addTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	if strings.TrimSpace(req.TrackID) == "" {
		response.WriteError(w, apperrors.NewBadRequest("track_id is required"))
		return
	}
	if req.Position <= 0 {
		response.WriteError(w, apperrors.NewBadRequest("position must be greater than 0"))
		return
	}

	if err := h.service.AddTrack(r.Context(), authservice.CurrentUser(r.Context()), playlistID, req.TrackID, req.Position); err != nil {
		writePlaylistError(w, err, "failed to add track")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) RemoveTrack(w http.ResponseWriter, r *http.Request) {
	playlistID := chi.URLParam(r, "id")
	trackID := chi.URLParam(r, "track_id")
	if playlistID == "" || trackID == "" {
		response.WriteError(w, apperrors.NewBadRequest("playlist id and track id are required"))
		return
	}

	if err := h.service.RemoveTrack(r.Context(), authservice.CurrentUser(r.Context()), playlistID, trackID); err != nil {
		writePlaylistError(w, err, "failed to remove track")
		return
	}

	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	playlistID := chi.URLParam(r, "id")
	if playlistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("playlist id is required"))
		return
	}
	if err := h.service.Delete(r.Context(), authservice.CurrentUser(r.Context()), playlistID); err != nil {
		writePlaylistError(w, err, "failed to delete playlist")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) RenamePlaylist(w http.ResponseWriter, r *http.Request) {
	playlistID := chi.URLParam(r, "id")
	if playlistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("playlist id is required"))
		return
	}
	var req renamePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		response.WriteError(w, apperrors.NewBadRequest("name is required"))
		return
	}
	if err := h.service.Rename(r.Context(), authservice.CurrentUser(r.Context()), playlistID, req.Name); err != nil {
		writePlaylistError(w, err, "failed to rename playlist")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) UpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	playlistID := chi.URLParam(r, "id")
	if playlistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("playlist id is required"))
		return
	}
	var req updatePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" {
		response.WriteError(w, apperrors.NewBadRequest("name is required"))
		return
	}
	if err := h.service.Update(r.Context(), authservice.CurrentUser(r.Context()), playlistID, req.Name, req.Description); err != nil {
		writePlaylistError(w, err, "failed to update playlist")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func writePlaylistError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case authservice.ErrUnauthorized:
		response.WriteError(w, apperrors.AppError{Code: "unauthorized", Message: "login required", HTTPStatus: http.StatusUnauthorized})
	case authservice.ErrForbidden:
		response.WriteError(w, apperrors.AppError{Code: "forbidden", Message: err.Error(), HTTPStatus: http.StatusForbidden})
	default:
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			response.WriteError(w, apperrors.NewNotFound("playlist not found"))
			return
		}
		response.WriteError(w, apperrors.NewInternal(fallback))
	}
}

func parseListParams(r *http.Request) (playlistsservice.ListParams, error) {
	params := playlistsservice.ListParams{}
	query := r.URL.Query()

	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 0 {
			return params, errors.New("invalid limit")
		}
		params.Limit = limit
	}
	if raw := query.Get("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return params, errors.New("invalid offset")
		}
		params.Offset = offset
	}

	sortBy := strings.ToLower(query.Get("sort"))
	if sortBy == "" {
		sortBy = "name"
	}
	switch sortBy {
	case "name", "created_at":
		params.SortBy = sortBy
	default:
		return params, errors.New("invalid sort")
	}
	return params, nil
}
