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
	albumsservice "music-server/internal/modules/albums/service"
	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Service interface {
	List(ctx context.Context, params albumsservice.ListParams) ([]domain.Album, error)
	GetByID(ctx context.Context, id string) (domain.Album, error)
	ListByArtistID(ctx context.Context, artistID string, params albumsservice.ListParams) ([]domain.Album, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r)
	if err != nil {
		response.WriteError(w, apperrors.NewBadRequest(err.Error()))
		return
	}

	albums, err := h.service.List(r.Context(), params)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to list albums"))
		return
	}

	response.WriteOK(w, map[string]any{"data": albums})
}

func (h *Handler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.WriteError(w, apperrors.NewBadRequest("album id is required"))
		return
	}

	album, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteError(w, apperrors.NewNotFound("album not found"))
			return
		}
		response.WriteError(w, apperrors.NewInternal("failed to load album"))
		return
	}

	response.WriteOK(w, map[string]any{"data": album})
}

func (h *Handler) ListArtistAlbums(w http.ResponseWriter, r *http.Request) {
	artistID := chi.URLParam(r, "id")
	if artistID == "" {
		response.WriteError(w, apperrors.NewBadRequest("artist id is required"))
		return
	}

	params, err := parseListParams(r)
	if err != nil {
		response.WriteError(w, apperrors.NewBadRequest(err.Error()))
		return
	}

	albums, err := h.service.ListByArtistID(r.Context(), artistID, params)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to list artist albums"))
		return
	}

	response.WriteOK(w, map[string]any{"data": albums})
}

func parseListParams(r *http.Request) (albumsservice.ListParams, error) {
	params := albumsservice.ListParams{}
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
	case "name", "year", "created_at":
		params.SortBy = sortBy
	default:
		return params, errors.New("invalid sort")
	}

	return params, nil
}
