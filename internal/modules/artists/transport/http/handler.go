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
	artistsservice "music-server/internal/modules/artists/service"
	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Service interface {
	List(ctx context.Context, params artistsservice.ListParams) ([]domain.Artist, error)
	GetByID(ctx context.Context, id string) (domain.Artist, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListArtists(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r, map[string]struct{}{
		"name":       {},
		"created_at": {},
	})
	if err != nil {
		response.WriteError(w, apperrors.NewBadRequest(err.Error()))
		return
	}

	artists, err := h.service.List(r.Context(), params)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to list artists"))
		return
	}

	response.WriteOK(w, map[string]any{"data": artists})
}

func (h *Handler) GetArtist(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.WriteError(w, apperrors.NewBadRequest("artist id is required"))
		return
	}

	artist, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteError(w, apperrors.NewNotFound("artist not found"))
			return
		}
		response.WriteError(w, apperrors.NewInternal("failed to load artist"))
		return
	}

	response.WriteOK(w, map[string]any{"data": artist})
}

func parseListParams(r *http.Request, allowedSort map[string]struct{}) (artistsservice.ListParams, error) {
	params := artistsservice.ListParams{}
	query := r.URL.Query()

	limitRaw := query.Get("limit")
	if limitRaw != "" {
		limit, err := strconv.Atoi(limitRaw)
		if err != nil || limit < 0 {
			return params, errors.New("invalid limit")
		}
		params.Limit = limit
	}

	offsetRaw := query.Get("offset")
	if offsetRaw != "" {
		offset, err := strconv.Atoi(offsetRaw)
		if err != nil || offset < 0 {
			return params, errors.New("invalid offset")
		}
		params.Offset = offset
	}

	sortBy := strings.ToLower(query.Get("sort"))
	if sortBy == "" {
		sortBy = "name"
	}
	if _, ok := allowedSort[sortBy]; !ok {
		return params, errors.New("invalid sort")
	}
	params.SortBy = sortBy
	return params, nil
}
