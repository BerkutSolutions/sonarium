package http

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	coverservice "music-server/internal/modules/coverart/service"
	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Service interface {
	AlbumOriginal(ctx context.Context, albumID string, withPlaceholder bool) (string, string, error)
	ArtistOriginal(ctx context.Context, artistID string, withPlaceholder bool) (string, string, error)
	AlbumThumb(ctx context.Context, albumID string, size int, withPlaceholder bool) (string, string, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) AlbumCover(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "album_id")
	if strings.TrimSpace(albumID) == "" {
		response.WriteError(w, apperrors.NewBadRequest("album id is required"))
		return
	}

	path, mimeType, err := h.service.AlbumOriginal(r.Context(), albumID, true)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.serveImage(w, r, path, mimeType)
}

func (h *Handler) ArtistCover(w http.ResponseWriter, r *http.Request) {
	artistID := chi.URLParam(r, "artist_id")
	if strings.TrimSpace(artistID) == "" {
		response.WriteError(w, apperrors.NewBadRequest("artist id is required"))
		return
	}

	path, mimeType, err := h.service.ArtistOriginal(r.Context(), artistID, true)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.serveImage(w, r, path, mimeType)
}

func (h *Handler) AlbumThumb(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "album_id")
	if strings.TrimSpace(albumID) == "" {
		response.WriteError(w, apperrors.NewBadRequest("album id is required"))
		return
	}

	sizeRaw := chi.URLParam(r, "size")
	size, err := strconv.Atoi(sizeRaw)
	if err != nil {
		response.WriteError(w, apperrors.NewBadRequest("invalid thumb size"))
		return
	}

	path, mimeType, err := h.service.AlbumThumb(r.Context(), albumID, size, true)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.serveImage(w, r, path, mimeType)
}

func (h *Handler) serveImage(w http.ResponseWriter, r *http.Request, path, mimeType string) {
	file, err := os.Open(path)
	if err != nil {
		h.writeError(w, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		h.writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, coverservice.ErrAlbumNotFound):
		response.WriteError(w, apperrors.NewNotFound("album not found"))
	case errors.Is(err, coverservice.ErrArtistNotFound):
		response.WriteError(w, apperrors.NewNotFound("artist not found"))
	case strings.Contains(strings.ToLower(err.Error()), "unsupported thumb size"):
		response.WriteError(w, apperrors.NewBadRequest("unsupported thumb size"))
	default:
		response.WriteError(w, apperrors.NewInternal("failed to resolve cover"))
	}
}
