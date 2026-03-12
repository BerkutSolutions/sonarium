package http

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Service interface {
	GetForTrack(ctx context.Context, trackID string) ([]int, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetTrackWaveform(w http.ResponseWriter, r *http.Request) {
	trackID := strings.TrimSpace(chi.URLParam(r, "id"))
	if trackID == "" {
		response.WriteError(w, apperrors.NewBadRequest("track id is required"))
		return
	}
	values, err := h.service.GetForTrack(r.Context(), trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteError(w, apperrors.NewNotFound("track not found"))
			return
		}
		response.WriteError(w, apperrors.NewInternal("failed to load waveform"))
		return
	}
	response.WriteOK(w, map[string]any{
		"data": map[string]any{
			"track_id":  trackID,
			"amplitude": values,
		},
	})
}
