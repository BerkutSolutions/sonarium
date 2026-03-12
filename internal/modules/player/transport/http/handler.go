package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	authservice "music-server/internal/modules/auth/service"
	libraryservice "music-server/internal/modules/library/service"
	playerservice "music-server/internal/modules/player/service"
	playerstate "music-server/internal/modules/player/state"
	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Service interface {
	GetState(ctx context.Context) playerstate.PlaybackState
	SetState(ctx context.Context, next playerstate.PlaybackState) playerstate.PlaybackState
	ReplaceQueue(ctx context.Context, items []playerstate.QueueItem, position int, contextType string, contextID string) playerstate.PlaybackState
	AppendQueue(ctx context.Context, items []playerstate.QueueItem) playerstate.PlaybackState
	RemoveQueueItem(ctx context.Context, index int) (playerstate.PlaybackState, error)
	ClearQueue(ctx context.Context) playerstate.PlaybackState
	MoveQueueItem(ctx context.Context, from int, to int) (playerstate.PlaybackState, error)
	ShuffleQueue(ctx context.Context, enabled bool) playerstate.PlaybackState
	RecordPlayed(ctx context.Context, userID, trackID string, positionSeconds int, contextType, contextID string) error
}

type Handler struct {
	service Service
}

type replaceQueueRequest struct {
	Queue         []playerstate.QueueItem `json:"queue"`
	QueuePosition int                     `json:"queue_position"`
	ContextType   string                  `json:"context_type"`
	ContextID     string                  `json:"context_id"`
}

type appendQueueRequest struct {
	Items []playerstate.QueueItem `json:"items"`
}

type removeQueueRequest struct {
	Index int `json:"index"`
}

type moveQueueRequest struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type shuffleRequest struct {
	Enabled bool `json:"enabled"`
}

type playedRequest struct {
	TrackID         string `json:"track_id"`
	PositionSeconds int    `json:"position_seconds"`
	ContextType     string `json:"context_type"`
	ContextID       string `json:"context_id"`
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetState(w http.ResponseWriter, r *http.Request) {
	response.WriteOK(w, map[string]any{"data": h.service.GetState(r.Context())})
}

func (h *Handler) SetState(w http.ResponseWriter, r *http.Request) {
	var req playerstate.PlaybackState
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	state := h.service.SetState(r.Context(), req)
	response.WriteOK(w, map[string]any{"data": state})
}

func (h *Handler) ReplaceQueue(w http.ResponseWriter, r *http.Request) {
	var req replaceQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	state := h.service.ReplaceQueue(r.Context(), req.Queue, req.QueuePosition, req.ContextType, req.ContextID)
	response.WriteOK(w, map[string]any{"data": state})
}

func (h *Handler) AppendQueue(w http.ResponseWriter, r *http.Request) {
	var req appendQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	state := h.service.AppendQueue(r.Context(), req.Items)
	response.WriteOK(w, map[string]any{"data": state})
}

func (h *Handler) RemoveQueueItem(w http.ResponseWriter, r *http.Request) {
	var req removeQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	state, err := h.service.RemoveQueueItem(r.Context(), req.Index)
	if err != nil {
		if errors.Is(err, playerservice.ErrQueueIndexOutOfRange) {
			response.WriteError(w, apperrors.NewBadRequest("queue index out of range"))
			return
		}
		response.WriteError(w, apperrors.NewInternal("failed to remove queue item"))
		return
	}
	response.WriteOK(w, map[string]any{"data": state})
}

func (h *Handler) ClearQueue(w http.ResponseWriter, r *http.Request) {
	state := h.service.ClearQueue(r.Context())
	response.WriteOK(w, map[string]any{"data": state})
}

func (h *Handler) MoveQueueItem(w http.ResponseWriter, r *http.Request) {
	var req moveQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	state, err := h.service.MoveQueueItem(r.Context(), req.From, req.To)
	if err != nil {
		if errors.Is(err, playerservice.ErrQueueIndexOutOfRange) {
			response.WriteError(w, apperrors.NewBadRequest("queue index out of range"))
			return
		}
		response.WriteError(w, apperrors.NewInternal("failed to move queue item"))
		return
	}
	response.WriteOK(w, map[string]any{"data": state})
}

func (h *Handler) ShuffleQueue(w http.ResponseWriter, r *http.Request) {
	var req shuffleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	state := h.service.ShuffleQueue(r.Context(), req.Enabled)
	response.WriteOK(w, map[string]any{"data": state})
}

func (h *Handler) Played(w http.ResponseWriter, r *http.Request) {
	var req playedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	if req.TrackID == "" {
		response.WriteError(w, apperrors.NewBadRequest("track_id is required"))
		return
	}
	if err := h.service.RecordPlayed(
		r.Context(),
		currentUserID(r.Context()),
		req.TrackID,
		req.PositionSeconds,
		req.ContextType,
		req.ContextID,
	); err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to record play event"))
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func currentUserID(ctx context.Context) string {
	if userID := authservice.CurrentUserID(ctx); userID != "" {
		return userID
	}
	return libraryservice.DefaultUserID
}
