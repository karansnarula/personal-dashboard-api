package widget

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	userAgent    = "personal-dashboard-api/0.1"
	maxBodyBytes = 1 << 20
	maxErrBody   = 200
)

// Client fetches live data for one widget type. config is the widget's
// stored (already validated) config JSON.
type Client interface {
	Fetch(ctx context.Context, config json.RawMessage) (any, error)
}

// ErrNotConfigured is returned when a provider needs an API key that was
// not supplied.
var ErrNotConfigured = errors.New("API key is not configured")

// UpstreamError is returned when a provider answers with a non-2xx status.
type UpstreamError struct {
	Provider string
	Status   int
	Body     string
}

func (e *UpstreamError) Error() string {
	if e.Status == http.StatusTooManyRequests {
		return fmt.Sprintf("%s rate limit exceeded (HTTP 429)", e.Provider)
	}
	return fmt.Sprintf("%s returned HTTP %d: %s", e.Provider, e.Status, e.Body)
}

func notConfigured(provider string) error {
	return fmt.Errorf("%s %w", provider, ErrNotConfigured)
}

// getJSON performs a GET, enforces a body limit, and decodes a 2xx JSON
// response into out. Timeouts and cancellation come from ctx.
func getJSON(ctx context.Context, hc *http.Client, provider, url string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%s: build request: %w", provider, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", provider, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return fmt.Errorf("%s: read response: %w", provider, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &UpstreamError{Provider: provider, Status: resp.StatusCode, Body: truncate(string(body), maxErrBody)}
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s: decode response: %w", provider, err)
	}
	return nil
}

func parseConfig[T any](raw json.RawMessage) (T, error) {
	var cfg T
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("stored config is unreadable: %w", err)
	}
	return cfg, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
