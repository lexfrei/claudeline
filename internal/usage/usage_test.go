package usage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lexfrei/claudeline/internal/keychain"
)

var errTest = errors.New("test error")

func TestParseBody(t *testing.T) {
	t.Parallel()

	resetsAt := time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)

	body := []byte(`{
		"five_hour": {"utilization": 30.5, "resets_at": "` + resetsAt + `"},
		"seven_day": {"utilization": 45.2, "resets_at": "` + resetsAt + `"}
	}`)

	data, err := ParseBody(body)
	if err != nil {
		t.Fatalf("ParseBody failed: %v", err)
	}

	if data.FiveHour == nil {
		t.Fatal("expected FiveHour to be set")
	}

	if data.SevenDay == nil {
		t.Fatal("expected SevenDay to be set")
	}

	if data.Extra != nil {
		t.Error("expected Extra to be nil")
	}

	if int(data.FiveHour.Utilization+halfRound) != 31 {
		t.Errorf("FiveHour utilization = %.1f, want ~30.5", data.FiveHour.Utilization)
	}
}

func TestParseBodyError(t *testing.T) {
	t.Parallel()

	body := []byte(`{"error": {"type": "authentication_error"}}`)

	data, err := ParseBody(body)
	if err != nil {
		t.Fatalf("ParseBody failed: %v", err)
	}

	if data.ErrorType != "authentication_error" {
		t.Errorf("ErrorType = %q, want %q", data.ErrorType, "authentication_error")
	}
}

func TestParseBodyExtra(t *testing.T) {
	t.Parallel()

	resetsAt := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)

	body := []byte(`{
		"five_hour": {"utilization": 10, "resets_at": "` + resetsAt + `"},
		"extra_usage": {
			"is_enabled": true,
			"monthly_limit": 5000,
			"used_credits": 128
		}
	}`)

	data, err := ParseBody(body)
	if err != nil {
		t.Fatalf("ParseBody failed: %v", err)
	}

	if data.Extra == nil {
		t.Fatal("expected Extra to be set")
	}

	if data.Extra.MonthlyLimit != 5000 {
		t.Errorf("MonthlyLimit = %.0f, want 5000", data.Extra.MonthlyLimit)
	}

	if data.Extra.UsedCredits != 128 {
		t.Errorf("UsedCredits = %.0f, want 128", data.Extra.UsedCredits)
	}
}

func TestParseBodyExtraDisabled(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"extra_usage": {
			"is_enabled": true,
			"monthly_limit": 5000,
			"used_credits": 0
		}
	}`)

	data, err := ParseBody(body)
	if err != nil {
		t.Fatalf("ParseBody failed: %v", err)
	}

	if data.Extra != nil {
		t.Error("Extra should be nil when used_credits is 0")
	}
}

func TestParseBodyInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseBody([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseWindowInvalidTimestamp(t *testing.T) {
	t.Parallel()

	win := &apiWindow{Utilization: 50, ResetsAt: "invalid"}
	got := parseWindow(win, 300)

	if got != nil {
		t.Error("expected nil for invalid timestamp")
	}
}

func TestParseWindowPastReset(t *testing.T) {
	t.Parallel()

	past := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	win := &apiWindow{Utilization: 50, ResetsAt: past}
	got := parseWindow(win, 300)

	if got == nil {
		t.Fatal("expected non-nil window")
	}

	if got.RemainingMinutes != 0 {
		t.Errorf("expected 0 remaining, got %d", got.RemainingMinutes)
	}
}

func TestFormatQuotaWindow(t *testing.T) {
	t.Parallel()

	win := &QuotaWindow{
		Utilization:      45.3,
		TotalMinutes:     10080,
		RemainingMinutes: 6857,
	}

	got := FormatQuotaWindow(win, "7d")
	if got == "" {
		t.Error("FormatQuotaWindow returned empty string")
	}

	if !strings.Contains(got, "7d") || !strings.Contains(got, "45%") {
		t.Errorf("FormatQuotaWindow = %q, missing expected content", got)
	}
}

// Tests below mutate package-level vars and cannot run in parallel.

func TestFetchCached(t *testing.T) {
	dir := t.TempDir()
	origPath := CachePath
	origToken := keychain.GetFn

	CachePath = filepath.Join(dir, "usage-cache.json")

	defer func() {
		CachePath = origPath
		keychain.GetFn = origToken
	}()

	resetsAt := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	cached := `{"five_hour":{"utilization":25,"resets_at":"` + resetsAt + `"}}`

	err := os.WriteFile(CachePath, []byte(cached), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	keychain.GetFn = func() (string, error) {
		t.Error("should not call GetFn when cache is valid")

		return "", errTest
	}

	data, err := Fetch()
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if data.FiveHour == nil {
		t.Error("expected FiveHour from cache")
	}
}

func TestFetchNoToken(t *testing.T) {
	dir := t.TempDir()
	origPath := CachePath
	origToken := keychain.GetFn

	CachePath = filepath.Join(dir, "usage-cache.json")

	defer func() {
		CachePath = origPath
		keychain.GetFn = origToken
	}()

	keychain.GetFn = func() (string, error) { return "", errTest }

	_, err := Fetch()
	if err == nil {
		t.Error("expected error when no token")
	}
}

func TestFetchHTTPError(t *testing.T) {
	dir := t.TempDir()
	origPath := CachePath
	origToken := keychain.GetFn
	origHTTP := HTTPGetFn

	CachePath = filepath.Join(dir, "usage-cache.json")

	defer func() {
		CachePath = origPath
		keychain.GetFn = origToken
		HTTPGetFn = origHTTP
	}()

	keychain.GetFn = func() (string, error) { return "test-token", nil }
	HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return nil, errTest
	}

	_, err := Fetch()
	if err == nil {
		t.Error("expected error on HTTP failure")
	}
}

func TestFetchSuccess(t *testing.T) {
	dir := t.TempDir()
	origPath := CachePath
	origToken := keychain.GetFn
	origHTTP := HTTPGetFn

	CachePath = filepath.Join(dir, "usage-cache.json")

	defer func() {
		CachePath = origPath
		keychain.GetFn = origToken
		HTTPGetFn = origHTTP
	}()

	resetsAt := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)

	keychain.GetFn = func() (string, error) { return "test-token", nil }
	HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return []byte(`{"five_hour":{"utilization":30,"resets_at":"` + resetsAt + `"}}`), nil
	}

	data, err := Fetch()
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if data.FiveHour == nil {
		t.Error("expected FiveHour to be set")
	}

	if _, statErr := os.Stat(CachePath); statErr != nil {
		t.Error("expected cache file to be written")
	}
}
