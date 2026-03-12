package http

import (
	"errors"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	streamservice "music-server/internal/modules/stream/service"
	transcodingservice "music-server/internal/modules/transcoding/service"
	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

var uuidPattern = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)

type Handler struct {
	logger     *zap.Logger
	service    *streamservice.Service
	transcoder *transcodingservice.Service
}

func NewHandler(logger *zap.Logger, service *streamservice.Service, transcoder *transcodingservice.Service) *Handler {
	return &Handler{
		logger:     logger,
		service:    service,
		transcoder: transcoder,
	}
}

func (h *Handler) StreamTrack(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	trackID := chi.URLParam(r, "track_id")
	if trackID == "" {
		response.WriteError(w, apperrors.NewBadRequest("track id is required"))
		return
	}
	if !uuidPattern.MatchString(trackID) {
		response.WriteError(w, apperrors.NewBadRequest("invalid track id"))
		return
	}

	streamTrack, err := h.service.ResolveTrack(r.Context(), trackID)
	if err != nil {
		h.writeStreamError(w, err)
		return
	}

	file, err := os.Open(streamTrack.FilePath)
	if err != nil {
		h.writeStreamError(w, err)
		return
	}
	defer file.Close()

	requestedFormat := r.URL.Query().Get("format")
	requestedBitrate, _ := strconv.Atoi(r.URL.Query().Get("bitrate"))
	if h.transcoder != nil && h.transcoder.ShouldTranscode(requestedFormat, requestedBitrate) {
		reader, mimeType, err := h.transcoder.OpenReader(r.Context(), transcodingservice.Request{
			InputPath: streamTrack.FilePath,
			Format:    requestedFormat,
			Bitrate:   requestedBitrate,
		})
		if err != nil {
			h.logger.Error("transcode failed", zap.Error(err), zap.String("track_id", streamTrack.TrackID))
			response.WriteError(w, apperrors.NewInternal("failed to transcode track"))
			return
		}
		defer reader.Close()
		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Transfer-Encoding", "chunked")
		_, _ = io.Copy(w, reader)
		h.logger.Info("transcoded stream served",
			zap.String("track_id", streamTrack.TrackID),
			zap.String("format", requestedFormat),
			zap.Int("bitrate", requestedBitrate),
			zap.String("remote_ip", remoteIP(r)),
			zap.Duration("duration", time.Since(start)),
		)
		return
	}

	w.Header().Set("Content-Type", streamTrack.MIMEType)
	w.Header().Set("Accept-Ranges", "bytes")

	http.ServeContent(w, r, streamTrack.FileName, streamTrack.ModTime, file)

	h.logger.Info("stream served",
		zap.String("track_id", streamTrack.TrackID),
		zap.String("remote_ip", remoteIP(r)),
		zap.Duration("duration", time.Since(start)),
	)
}

func (h *Handler) writeStreamError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, streamservice.ErrTrackNotFound):
		response.WriteError(w, apperrors.NewNotFound("track not found"))
	case errors.Is(err, streamservice.ErrFileNotFound), errors.Is(err, os.ErrNotExist):
		response.WriteError(w, apperrors.NewNotFound("track file not found"))
	default:
		h.logger.Error("stream failed", zap.Error(err))
		response.WriteError(w, apperrors.NewInternal("failed to stream track"))
	}
}

func remoteIP(r *http.Request) string {
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}
