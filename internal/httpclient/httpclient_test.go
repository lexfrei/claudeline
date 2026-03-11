package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") != "hello" {
			t.Errorf("expected X-Test header, got %q", r.Header.Get("X-Test"))
		}

		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	resp, err := Get(srv.URL, map[string]string{"X-Test": "hello"}, 5*time.Second)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(resp.Body) != `{"ok":true}` {
		t.Errorf("unexpected body: %q", resp.Body)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

func TestGetReturnsStatusCode(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(http.StatusTooManyRequests)

		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error"}}`))
	}))
	defer srv.Close()

	resp, err := Get(srv.URL, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Retry-After") != "10" {
		t.Errorf("expected Retry-After header, got %q", resp.Header.Get("Retry-After"))
	}
}

func TestGetNoHeaders(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	resp, err := Get(srv.URL, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(resp.Body) != "ok" {
		t.Errorf("unexpected body: %q", resp.Body)
	}
}

func TestGetError(t *testing.T) {
	t.Parallel()

	_, err := Get("http://127.0.0.1:1", nil, 1*time.Second)
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}
