package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	authservice "music-server/internal/modules/auth/service"
	sharingservice "music-server/internal/modules/sharing/service"
	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Handler struct {
	service *sharingservice.Service
}

type userShareRequest struct {
	UserID     string `json:"user_id"`
	Permission string `json:"permission"`
}

type publicShareRequest struct {
	Enabled bool `json:"enabled"`
}

func NewHandler(service *sharingservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListEntityShares(w http.ResponseWriter, r *http.Request) {
	entityType, entityID, ok := shareRouteParams(w, r)
	if !ok {
		return
	}
	shares, err := h.service.ListEntityShares(r.Context(), authservice.CurrentUser(r.Context()), entityType, entityID)
	if err != nil {
		writeShareError(w, err, "failed to list shares")
		return
	}
	response.WriteOK(w, map[string]any{"data": shares})
}

func (h *Handler) ShareWithUser(w http.ResponseWriter, r *http.Request) {
	entityType, entityID, ok := shareRouteParams(w, r)
	if !ok {
		return
	}
	var req userShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	share, err := h.service.UpsertUserShare(
		r.Context(),
		authservice.CurrentUser(r.Context()),
		entityType,
		entityID,
		strings.TrimSpace(req.UserID),
		strings.TrimSpace(req.Permission),
	)
	if err != nil {
		writeShareError(w, err, "failed to share entity")
		return
	}
	response.WriteOK(w, map[string]any{"data": share})
}

func (h *Handler) SetPublicShare(w http.ResponseWriter, r *http.Request) {
	entityType, entityID, ok := shareRouteParams(w, r)
	if !ok {
		return
	}
	var req publicShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	share, err := h.service.SetPublicShare(r.Context(), authservice.CurrentUser(r.Context()), entityType, entityID, req.Enabled)
	if err != nil {
		writeShareError(w, err, "failed to update public share")
		return
	}
	response.WriteOK(w, map[string]any{"data": share})
}

func (h *Handler) DeleteShare(w http.ResponseWriter, r *http.Request) {
	shareID := strings.TrimSpace(chi.URLParam(r, "share_id"))
	if shareID == "" {
		response.WriteError(w, apperrors.NewBadRequest("share id is required"))
		return
	}
	if err := h.service.DeleteShare(r.Context(), authservice.CurrentUser(r.Context()), shareID); err != nil {
		writeShareError(w, err, "failed to delete share")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) ListReceivedShares(w http.ResponseWriter, r *http.Request) {
	current := authservice.CurrentUser(r.Context())
	if current == nil {
		writeShareError(w, authservice.ErrUnauthorized, "failed to list received shares")
		return
	}
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID == "" {
		userID = current.ID
	}
	shares, err := h.service.ListReceivedShares(r.Context(), current, userID)
	if err != nil {
		writeShareError(w, err, "failed to list received shares")
		return
	}
	response.WriteOK(w, map[string]any{"data": shares})
}

func shareRouteParams(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	entityType := strings.TrimSpace(strings.ToLower(chi.URLParam(r, "entity_type")))
	entityID := strings.TrimSpace(chi.URLParam(r, "entity_id"))
	if entityType == "" || entityID == "" {
		response.WriteError(w, apperrors.NewBadRequest("entity type and id are required"))
		return "", "", false
	}
	return entityType, entityID, true
}

func writeShareError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case authservice.ErrUnauthorized:
		response.WriteError(w, apperrors.AppError{Code: "unauthorized", Message: "login required", HTTPStatus: http.StatusUnauthorized})
	case authservice.ErrForbidden:
		response.WriteError(w, apperrors.AppError{Code: "forbidden", Message: err.Error(), HTTPStatus: http.StatusForbidden})
	case sharingservice.ErrInvalidPermission:
		response.WriteError(w, apperrors.NewBadRequest("invalid permission"))
	default:
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			response.WriteError(w, apperrors.NewNotFound("share not found"))
			return
		}
		response.WriteError(w, apperrors.NewInternal(fallback))
	}
}
