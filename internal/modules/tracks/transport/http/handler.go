package http

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"music-server/internal/domain"
	tracksservice "music-server/internal/modules/tracks/service"
	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Service interface {
	List(ctx context.Context, params tracksservice.ListParams) ([]domain.Track, error)
	GetByID(ctx context.Context, id string) (domain.Track, error)
	ListByAlbumID(ctx context.Context, albumID string, params tracksservice.ListParams) ([]domain.Track, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListTracks(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r)
	if err != nil {
		response.WriteError(w, apperrors.NewBadRequest(err.Error()))
		return
	}

	tracks, err := h.service.List(r.Context(), params)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to list tracks"))
		return
	}

	response.WriteOK(w, map[string]any{"data": tracks})
}

func (h *Handler) GetTrack(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.WriteError(w, apperrors.NewBadRequest("track id is required"))
		return
	}

	track, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteError(w, apperrors.NewNotFound("track not found"))
			return
		}
		response.WriteError(w, apperrors.NewInternal("failed to load track"))
		return
	}

	response.WriteOK(w, map[string]any{"data": track})
}

func (h *Handler) ListAlbumTracks(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	if albumID == "" {
		response.WriteError(w, apperrors.NewBadRequest("album id is required"))
		return
	}

	params, err := parseListParams(r)
	if err != nil {
		response.WriteError(w, apperrors.NewBadRequest(err.Error()))
		return
	}

	tracks, err := h.service.ListByAlbumID(r.Context(), albumID, params)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to list album tracks"))
		return
	}

	response.WriteOK(w, map[string]any{"data": tracks})
}

func parseListParams(r *http.Request) (tracksservice.ListParams, error) {
	params := tracksservice.ListParams{}
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
