// Package httpclient provides a simple HTTP GET function for statusline API calls.
package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GetFn is the type for HTTP GET functions.
type GetFn func(url string, headers map[string]string, timeout time.Duration) ([]byte, error)

// Get performs an HTTP GET request with context timeout.
func Get(url string, headers map[string]string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	for key, val := range headers {
		req.Header.Set(key, val)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return body, nil
}
