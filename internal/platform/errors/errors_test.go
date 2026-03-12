package errors

import "testing"

func TestNewBadRequest(t *testing.T) {
	err := NewBadRequest("invalid input")

	if err.Code != "invalid_request" {
		t.Fatalf("expected code invalid_request, got %q", err.Code)
	}
	if err.Message != "invalid input" {
		t.Fatalf("expected message invalid input, got %q", err.Message)
	}
	if err.HTTPStatus != 400 {
		t.Fatalf("expected status 400, got %d", err.HTTPStatus)
	}
}

func TestNewNotFound(t *testing.T) {
	err := NewNotFound("track not found")

	if err.Code != "not_found" {
		t.Fatalf("expected code not_found, got %q", err.Code)
	}
	if err.Message != "track not found" {
		t.Fatalf("expected message track not found, got %q", err.Message)
	}
	if err.HTTPStatus != 404 {
		t.Fatalf("expected status 404, got %d", err.HTTPStatus)
	}
}

func TestNewInternal(t *testing.T) {
	err := NewInternal("unexpected failure")

	if err.Code != "internal_error" {
		t.Fatalf("expected code internal_error, got %q", err.Code)
	}
	if err.Message != "unexpected failure" {
		t.Fatalf("expected message unexpected failure, got %q", err.Message)
	}
	if err.HTTPStatus != 500 {
		t.Fatalf("expected status 500, got %d", err.HTTPStatus)
	}
}
