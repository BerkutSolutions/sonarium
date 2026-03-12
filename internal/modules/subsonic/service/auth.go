package service

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"strings"
)

var (
	ErrMissingAuthParams      = errors.New("missing auth params")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrUnsupportedAPIVersion  = errors.New("unsupported api version")
	ErrMissingProtocolParams  = errors.New("missing protocol params")
	ErrUnsupportedFormatParam = errors.New("unsupported format")
)

type AuthConfig struct {
	Username   string
	Password   string
	MinVersion string
}

type ProtocolParams struct {
	Username string
	Token    string
	Salt     string
	Version  string
	Client   string
	Format   string
}

type Authenticator struct {
	cfg AuthConfig
}

func NewAuthenticator(cfg AuthConfig) *Authenticator {
	return &Authenticator{cfg: cfg}
}

func (a *Authenticator) Validate(params ProtocolParams) error {
	if strings.TrimSpace(params.Username) == "" ||
		strings.TrimSpace(params.Token) == "" ||
		strings.TrimSpace(params.Salt) == "" {
		return ErrMissingAuthParams
	}
	if strings.TrimSpace(params.Version) == "" ||
		strings.TrimSpace(params.Client) == "" ||
		strings.TrimSpace(params.Format) == "" {
		return ErrMissingProtocolParams
	}
	if params.Format != "json" && params.Format != "xml" {
		return ErrUnsupportedFormatParam
	}

	if compareVersion(params.Version, a.cfg.MinVersion) < 0 {
		return ErrUnsupportedAPIVersion
	}

	if params.Username != a.cfg.Username {
		return ErrInvalidCredentials
	}

	hash := md5.Sum([]byte(a.cfg.Password + params.Salt))
	expected := hex.EncodeToString(hash[:])
	if strings.ToLower(params.Token) != expected {
		return ErrInvalidCredentials
	}

	return nil
}

func compareVersion(a, b string) int {
	pa := parseVersion(a)
	pb := parseVersion(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(v string) [3]int {
	var out [3]int
	parts := strings.Split(v, ".")
	for i := 0; i < len(parts) && i < 3; i++ {
		n := 0
		for _, ch := range parts[i] {
			if ch < '0' || ch > '9' {
				break
			}
			n = n*10 + int(ch-'0')
		}
		out[i] = n
	}
	return out
}
