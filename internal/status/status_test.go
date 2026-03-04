package status

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func TestFetchAlertCached(t *testing.T) {
	dir := t.TempDir()
	origPath := CachePath
	CachePath = filepath.Join(dir, "status-cache.json")

	defer func() { CachePath = origPath }()

	err := os.WriteFile(CachePath, []byte("🔶 major outage"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	got := FetchAlert()
	if got != "🔶 major outage" {
		t.Errorf("expected cached alert, got %q", got)
	}
}

func TestFetchAlertFromAPI(t *testing.T) {
	dir := t.TempDir()
	origPath := CachePath
	origHTTP := HTTPGetFn

	CachePath = filepath.Join(dir, "status-cache.json")

	defer func() {
		CachePath = origPath
		HTTPGetFn = origHTTP
	}()

	HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return []byte(`{"status":{"indicator":"minor"}}`), nil
	}

	got := FetchAlert()
	if got != "⚠️ degraded" {
		t.Errorf("expected degraded, got %q", got)
	}
}

func TestFetchAlertNone(t *testing.T) {
	dir := t.TempDir()
	origPath := CachePath
	origHTTP := HTTPGetFn

	CachePath = filepath.Join(dir, "status-cache.json")

	defer func() {
		CachePath = origPath
		HTTPGetFn = origHTTP
	}()

	HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return []byte(`{"status":{"indicator":"none"}}`), nil
	}

	got := FetchAlert()
	if got != "" {
		t.Errorf("expected empty for none indicator, got %q", got)
	}
}

func TestFetchAlertHTTPError(t *testing.T) {
	dir := t.TempDir()
	origPath := CachePath
	origHTTP := HTTPGetFn

	CachePath = filepath.Join(dir, "status-cache.json")

	defer func() {
		CachePath = origPath
		HTTPGetFn = origHTTP
	}()

	HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return nil, errTest
	}

	got := FetchAlert()
	if got != "" {
		t.Errorf("expected empty on HTTP error, got %q", got)
	}
}

func TestFetchAlertBadJSON(t *testing.T) {
	dir := t.TempDir()
	origPath := CachePath
	origHTTP := HTTPGetFn

	CachePath = filepath.Join(dir, "status-cache.json")

	defer func() {
		CachePath = origPath
		HTTPGetFn = origHTTP
	}()

	HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return []byte(`not json`), nil
	}

	got := FetchAlert()
	if got != "" {
		t.Errorf("expected empty on bad JSON, got %q", got)
	}
}
