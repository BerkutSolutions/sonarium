package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	searchservice "music-server/internal/modules/search/service"
	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Service interface {
	Search(ctx context.Context, params searchservice.Params) (searchservice.Result, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	params, err := parseParams(r)
	if err != nil {
		response.WriteError(w, apperrors.NewBadRequest(err.Error()))
		return
	}

	result, err := h.service.Search(r.Context(), params)
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("search failed"))
		return
	}

	response.WriteOK(w, map[string]any{"data": result})
}

func parseParams(r *http.Request) (searchservice.Params, error) {
	params := searchservice.Params{}
	query := r.URL.Query()

	params.Query = strings.TrimSpace(query.Get("q"))
	if params.Query == "" {
		return params, errors.New("q is required")
	}

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
