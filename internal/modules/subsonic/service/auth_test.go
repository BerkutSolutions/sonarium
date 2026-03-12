package service

import (
	"crypto/md5"
	"encoding/hex"
	"testing"
)

func TestAuthenticatorValidateSuccess(t *testing.T) {
	auth := NewAuthenticator(AuthConfig{
		Username:   "admin",
		Password:   "secret",
		MinVersion: "1.16.1",
	})
	salt := "abc123"
	hash := md5.Sum([]byte("secret" + salt))
	token := hex.EncodeToString(hash[:])

	err := auth.Validate(ProtocolParams{
		Username: "admin",
		Token:    token,
		Salt:     salt,
		Version:  "1.16.1",
		Client:   "test-client",
		Format:   "json",
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestAuthenticatorValidateInvalidToken(t *testing.T) {
	auth := NewAuthenticator(AuthConfig{
		Username:   "admin",
		Password:   "secret",
		MinVersion: "1.16.1",
	})
	err := auth.Validate(ProtocolParams{
		Username: "admin",
		Token:    "wrong",
		Salt:     "abc",
		Version:  "1.16.1",
		Client:   "test-client",
		Format:   "json",
	})
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
