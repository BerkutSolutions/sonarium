package response

import (
	"encoding/json"
	"net/http"

	apperrors "music-server/internal/platform/errors"
)

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteOK(w http.ResponseWriter, payload any) {
	WriteJSON(w, http.StatusOK, payload)
}

func WriteError(w http.ResponseWriter, err apperrors.AppError) {
	body := errorEnvelope{}
	body.Error.Code = err.Code
	body.Error.Message = err.Message

	if body.Error.Code == "" {
		body.Error.Code = "internal_error"
	}
	if body.Error.Message == "" {
		body.Error.Message = "internal server error"
	}
	if err.HTTPStatus == 0 {
		err.HTTPStatus = http.StatusInternalServerError
	}

	WriteJSON(w, err.HTTPStatus, body)
}
