package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	authservice "music-server/internal/modules/auth/service"
	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Handler struct {
	service *authservice.Service
}

type credentialsRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type activeRequest struct {
	Active bool `json:"active"`
}

type registrationRequest struct {
	Open bool `json:"open"`
}

type profileUpdateRequest struct {
	Username      string `json:"username"`
	DisplayName   string `json:"display_name"`
	ProfilePublic bool   `json:"profile_public"`
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func NewHandler(service *authservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.Status(r.Context(), sessionTokenFromRequest(r))
	if err != nil {
		response.WriteError(w, apperrors.NewInternal("failed to load auth status"))
		return
	}
	response.WriteOK(w, map[string]any{"data": status})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	user, session, err := h.service.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeAuthError(w, err, "failed to login")
		return
	}
	writeSessionCookie(w, session)
	response.WriteOK(w, map[string]any{"data": map[string]any{"user": user}})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	displayName := req.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = req.Username
	}
	user, session, err := h.service.Register(r.Context(), req.Username, displayName, req.Password)
	if err != nil {
		writeAuthError(w, err, "failed to register")
		return
	}
	writeSessionCookie(w, session)
	response.WriteOK(w, map[string]any{"data": map[string]any{"user": user}})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	_ = h.service.Logout(r.Context(), sessionTokenFromRequest(r))
	clearSessionCookie(w)
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, open, err := h.service.ListUsers(r.Context(), authservice.CurrentUser(r.Context()))
	if err != nil {
		writeAuthError(w, err, "failed to list users")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]any{"users": users, "registration_open": open}})
}

func (h *Handler) ListShareableUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListShareableUsers(r.Context(), authservice.CurrentUser(r.Context()))
	if err != nil {
		writeAuthError(w, err, "failed to list shareable users")
		return
	}
	response.WriteOK(w, map[string]any{"data": users})
}

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(chi.URLParam(r, "user_id"))
	if userID == "" {
		response.WriteError(w, apperrors.NewBadRequest("user id is required"))
		return
	}
	var req activeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	if err := h.service.SetUserActive(r.Context(), authservice.CurrentUser(r.Context()), userID, req.Active); err != nil {
		writeAuthError(w, err, "failed to update user")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(chi.URLParam(r, "user_id"))
	if userID == "" {
		response.WriteError(w, apperrors.NewBadRequest("user id is required"))
		return
	}
	if err := h.service.DeleteUser(r.Context(), authservice.CurrentUser(r.Context()), userID); err != nil {
		writeAuthError(w, err, "failed to delete user")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) SetRegistrationOpen(w http.ResponseWriter, r *http.Request) {
	var req registrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	if err := h.service.SetRegistrationOpen(r.Context(), authservice.CurrentUser(r.Context()), req.Open); err != nil {
		writeAuthError(w, err, "failed to update registration setting")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req profileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	user, err := h.service.UpdateProfile(r.Context(), authservice.CurrentUser(r.Context()), req.Username, req.DisplayName, req.ProfilePublic)
	if err != nil {
		writeAuthError(w, err, "failed to update profile")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]any{"user": user}})
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(chi.URLParam(r, "user_id"))
	if userID == "" {
		current := authservice.CurrentUser(r.Context())
		if current == nil {
			writeAuthError(w, authservice.ErrUnauthorized, "failed to load profile")
			return
		}
		userID = current.ID
	}
	user, err := h.service.GetProfile(r.Context(), authservice.CurrentUser(r.Context()), userID)
	if err != nil {
		writeAuthError(w, err, "failed to load profile")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]any{"user": user}})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req passwordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid request body"))
		return
	}
	if err := h.service.ChangePassword(r.Context(), authservice.CurrentUser(r.Context()), req.CurrentPassword, req.NewPassword); err != nil {
		writeAuthError(w, err, "failed to change password")
		return
	}
	response.WriteOK(w, map[string]any{"data": map[string]bool{"ok": true}})
}

func sessionTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(authservice.SessionCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func writeSessionCookie(w http.ResponseWriter, session authservice.Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     authservice.SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authservice.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func writeAuthError(w http.ResponseWriter, err error, fallback string) {
	switch err {
	case authservice.ErrInvalidCredentials:
		response.WriteError(w, apperrors.AppError{Code: "unauthorized", Message: "invalid username or password", HTTPStatus: http.StatusUnauthorized})
	case authservice.ErrRegistrationClosed:
		response.WriteError(w, apperrors.AppError{Code: "registration_closed", Message: "registration is closed", HTTPStatus: http.StatusForbidden})
	case authservice.ErrUserExists:
		response.WriteError(w, apperrors.AppError{Code: "user_exists", Message: "user already exists", HTTPStatus: http.StatusConflict})
	case authservice.ErrUnauthorized:
		response.WriteError(w, apperrors.AppError{Code: "unauthorized", Message: "login required", HTTPStatus: http.StatusUnauthorized})
	case authservice.ErrForbidden, authservice.ErrLastAdmin:
		response.WriteError(w, apperrors.AppError{Code: "forbidden", Message: err.Error(), HTTPStatus: http.StatusForbidden})
	default:
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			response.WriteError(w, apperrors.NewNotFound("user not found"))
			return
		}
		response.WriteError(w, apperrors.NewInternal(fallback))
	}
}
