package http

import (
	"net/http"
	"strings"

	authservice "music-server/internal/modules/auth/service"
	"music-server/internal/platform/http/response"
)

func AuthRequired(service *authservice.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isAuthExempt(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			cookie, err := r.Cookie(authservice.SessionCookieName)
			if err != nil || strings.TrimSpace(cookie.Value) == "" {
				writeUnauthorized(w)
				return
			}
			_, user, err := service.ValidateSession(r.Context(), cookie.Value)
			if err != nil || user == nil {
				writeUnauthorized(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(authservice.WithCurrentUser(r.Context(), user)))
		})
	}
}

func isAuthExempt(path string) bool {
	return path == "/api/auth/status" || path == "/api/auth/login" || path == "/api/auth/register"
}

func writeUnauthorized(w http.ResponseWriter) {
	response.WriteJSON(w, http.StatusUnauthorized, map[string]any{
		"error": map[string]string{
			"code":    "unauthorized",
			"message": "login required",
		},
	})
}
