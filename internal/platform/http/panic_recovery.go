package http

import (
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"

	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

func PanicRecovery(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						zap.Any("panic", rec),
						zap.ByteString("stack", debug.Stack()),
						zap.String("request_id", GetRequestID(r.Context())),
						zap.String("path", r.URL.Path),
					)
					response.WriteError(w, apperrors.NewInternal("internal server error"))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
