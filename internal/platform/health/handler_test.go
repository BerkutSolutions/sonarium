package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDependency struct {
	name string
	err  error
}

func (f fakeDependency) Name() string {
	return f.name
}

func (f fakeDependency) Check(_ context.Context) error {
	return f.err
}

func TestHealthzHandler(t *testing.T) {
	service := NewService()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	service.HealthzHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	if body != `{"status":"ok"}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestReadyzHandlerReady(t *testing.T) {
	service := NewService(fakeDependency{name: "postgres"})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	service.ReadyzHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	if body != `{"status":"ready"}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestReadyzHandlerNotReady(t *testing.T) {
	service := NewService(fakeDependency{name: "postgres", err: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	service.ReadyzHandler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	expected := `{"error":{"code":"not_ready","message":"postgres is not ready"}}`
	if body != expected {
		t.Fatalf("unexpected body: %s", body)
	}
}
