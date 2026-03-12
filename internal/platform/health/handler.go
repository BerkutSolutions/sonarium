package health

import (
	"context"
	"net/http"
	"time"

	apperrors "music-server/internal/platform/errors"
	"music-server/internal/platform/http/response"
)

type Dependency interface {
	Name() string
	Check(ctx context.Context) error
}

type Service struct {
	dependencies []Dependency
}

func NewService(dependencies ...Dependency) *Service {
	return &Service{dependencies: dependencies}
}

func (s *Service) HealthzHandler(w http.ResponseWriter, _ *http.Request) {
	response.WriteOK(w, map[string]string{"status": "ok"})
}

func (s *Service) ReadyzHandler(w http.ResponseWriter, r *http.Request) {
	checkCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	for _, dep := range s.dependencies {
		if err := dep.Check(checkCtx); err != nil {
			response.WriteError(w, apperrors.AppError{
				Code:       "not_ready",
				Message:    dep.Name() + " is not ready",
				HTTPStatus: http.StatusServiceUnavailable,
			})
			return
		}
	}

	response.WriteOK(w, map[string]string{"status": "ready"})
}
