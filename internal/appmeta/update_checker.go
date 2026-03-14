package appmeta

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type UpdateCheckResult struct {
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
	ReleaseURL     string    `json:"release_url"`
	HasUpdate      bool      `json:"has_update"`
	CheckedAt      time.Time `json:"checked_at"`
	Source         string    `json:"source"`
}

type UpdateChecker struct {
	client      *http.Client
	repository  string
	releasesURL string

	mu          sync.RWMutex
	cached      *UpdateCheckResult
	lastRequest time.Time
	nextAllowed time.Time
}

func NewUpdateChecker(repositoryURL, releasesURL string) *UpdateChecker {
	repositoryURL = strings.TrimSpace(repositoryURL)
	releasesURL = strings.TrimSpace(releasesURL)
	if repositoryURL == "" {
		repositoryURL = DefaultRepositoryURL
	}
	if releasesURL == "" {
		releasesURL = DefaultGitHubAPIReleases
	}
	return &UpdateChecker{
		client:      &http.Client{Timeout: 5 * time.Second},
		repository:  repositoryURL,
		releasesURL: releasesURL,
	}
}

func (c *UpdateChecker) LastResult() *UpdateCheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.cached == nil {
		return nil
	}
	copy := *c.cached
	return &copy
}

func (c *UpdateChecker) Check(ctx context.Context, currentVersion string) (*UpdateCheckResult, error) {
	now := time.Now().UTC()

	c.mu.RLock()
	nextAllowed := c.nextAllowed
	lastRequest := c.lastRequest
	cached := c.cached
	c.mu.RUnlock()

	if cached != nil && now.Sub(lastRequest) < 30*time.Minute {
		copy := *cached
		return &copy, nil
	}
	if now.Before(nextAllowed) && cached != nil {
		copy := *cached
		return &copy, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.releasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "sonarium/"+currentVersion)

	resp, err := c.client.Do(req)
	if err != nil {
		return c.fallbackUnavailable(now, currentVersion), nil
	}
	defer resp.Body.Close()

	type releasePayload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}

	var payload releasePayload
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return c.LastResult(), err
		}
	} else {
		c.setNextAllowedFromHeaders(resp.Header, now)
		return c.fallbackUnavailable(now, currentVersion), nil
	}

	latest := normalizeVersion(payload.TagName)
	current := normalizeVersion(currentVersion)
	result := &UpdateCheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  payload.TagName,
		ReleaseURL:     payload.HTMLURL,
		HasUpdate:      compareSemver(latest, current) > 0,
		CheckedAt:      now,
		Source:         "github",
	}
	c.mu.Lock()
	c.cached = result
	c.lastRequest = now
	c.nextAllowed = now
	c.mu.Unlock()
	return c.LastResult(), nil
}

func (c *UpdateChecker) fallbackUnavailable(now time.Time, currentVersion string) *UpdateCheckResult {
	last := c.LastResult()
	if last != nil {
		return last
	}
	fallback := &UpdateCheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  currentVersion,
		ReleaseURL:     c.repository,
		HasUpdate:      false,
		CheckedAt:      now,
		Source:         "github_unavailable",
	}
	c.mu.Lock()
	c.cached = fallback
	c.lastRequest = now
	if c.nextAllowed.Before(now) {
		c.nextAllowed = now.Add(15 * time.Minute)
	}
	c.mu.Unlock()
	return c.LastResult()
}

func (c *UpdateChecker) setNextAllowedFromHeaders(headers http.Header, now time.Time) {
	retryAfter := strings.TrimSpace(headers.Get("Retry-After"))
	if retryAfter != "" {
		if sec, err := strconv.Atoi(retryAfter); err == nil && sec > 0 {
			c.mu.Lock()
			c.nextAllowed = now.Add(time.Duration(sec) * time.Second)
			c.lastRequest = now
			c.mu.Unlock()
			return
		}
	}

	reset := strings.TrimSpace(headers.Get("X-RateLimit-Reset"))
	if reset != "" {
		if unixSec, err := strconv.ParseInt(reset, 10, 64); err == nil && unixSec > 0 {
			next := time.Unix(unixSec, 0).UTC()
			c.mu.Lock()
			c.nextAllowed = next
			c.lastRequest = now
			c.mu.Unlock()
			return
		}
	}

	c.mu.Lock()
	c.nextAllowed = now.Add(15 * time.Minute)
	c.lastRequest = now
	c.mu.Unlock()
}

func normalizeVersion(value string) string {
	result := strings.TrimSpace(strings.ToLower(value))
	return strings.TrimPrefix(result, "v")
}

func compareSemver(a, b string) int {
	parse := func(value string) [3]int {
		var out [3]int
		parts := strings.Split(value, ".")
		for i := 0; i < len(parts) && i < 3; i++ {
			n, _ := strconv.Atoi(strings.TrimSpace(parts[i]))
			out[i] = n
		}
		return out
	}
	av := parse(a)
	bv := parse(b)
	for i := 0; i < 3; i++ {
		if av[i] > bv[i] {
			return 1
		}
		if av[i] < bv[i] {
			return -1
		}
	}
	return 0
}
