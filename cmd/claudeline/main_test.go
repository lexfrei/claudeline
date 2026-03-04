package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lexfrei/claudeline/internal/keychain"
	"github.com/lexfrei/claudeline/internal/status"
	"github.com/lexfrei/claudeline/internal/usage"
)

func setupTestEnv(t *testing.T) func() {
	t.Helper()

	dir := t.TempDir()

	origStatusPath := status.CachePath
	origUsagePath := usage.CachePath
	origStatusHTTP := status.HTTPGetFn
	origUsageHTTP := usage.HTTPGetFn
	origToken := keychain.GetFn

	status.CachePath = filepath.Join(dir, "status-cache.json")
	usage.CachePath = filepath.Join(dir, "usage-cache.json")

	return func() {
		status.CachePath = origStatusPath
		usage.CachePath = origUsagePath
		status.HTTPGetFn = origStatusHTTP
		usage.HTTPGetFn = origUsageHTTP
		keychain.GetFn = origToken
	}
}

func failHTTP(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
	return nil, keychain.ErrNoToken
}

func TestBuildStatuslineMinimal(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = failHTTP
	usage.HTTPGetFn = failHTTP

	got := buildStatusline([]byte(`{}`))

	if !strings.Contains(got, "🤖 Claude") {
		t.Errorf("expected model name, got %q", got)
	}

	if !strings.Contains(got, "💰 $0.00") {
		t.Errorf("expected zero cost, got %q", got)
	}

	if !strings.Contains(got, "⏳") {
		t.Errorf("expected placeholder, got %q", got)
	}
}

func TestBuildStatuslineWithModel(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = failHTTP
	usage.HTTPGetFn = failHTTP

	input := `{"model":{"display_name":"Opus 4.6"},"cost":{"total_cost_usd":42.50}}`
	got := buildStatusline([]byte(input))

	if !strings.Contains(got, "🤖 Opus 4.6") {
		t.Errorf("expected Opus 4.6, got %q", got)
	}

	if !strings.Contains(got, "💰 $42.50") {
		t.Errorf("expected $42.50, got %q", got)
	}
}

func TestBuildStatuslineWithContext(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = failHTTP
	usage.HTTPGetFn = failHTTP

	input := `{"context_window":{"used_percentage":75}}`
	got := buildStatusline([]byte(input))

	if !strings.Contains(got, "🧠 75%") {
		t.Errorf("expected context percentage, got %q", got)
	}
}

func TestBuildStatuslineWithCompactions(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = failHTTP
	usage.HTTPGetFn = failHTTP

	transcript := filepath.Join(t.TempDir(), "transcript.jsonl")
	lines := "{\"subtype\":\"compact_boundary\"}\n{\"subtype\":\"compact_boundary\"}\n"

	err := os.WriteFile(transcript, []byte(lines), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	input := `{"transcript_path":"` + transcript + `"}`
	got := buildStatusline([]byte(input))

	if !strings.Contains(got, "🔄 2") {
		t.Errorf("expected compaction count, got %q", got)
	}
}

func TestBuildStatuslineWithStatusAlert(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return []byte(`{"status":{"indicator":"major"}}`), nil
	}
	usage.HTTPGetFn = failHTTP

	got := buildStatusline([]byte(`{}`))

	if !strings.Contains(got, "🔶 major outage") {
		t.Errorf("expected major outage alert, got %q", got)
	}
}

func TestBuildStatuslineInvalidJSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = failHTTP
	usage.HTTPGetFn = failHTTP

	got := buildStatusline([]byte(`not json`))

	if !strings.Contains(got, "🤖 Claude") {
		t.Errorf("expected graceful degradation, got %q", got)
	}
}

func TestAppendUsageSegmentsLoginNeeded(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "test-token", nil }
	usage.HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return []byte(`{"error":{"type":"authentication_error"}}`), nil
	}

	segments := appendUsageSegments(nil)
	joined := strings.Join(segments, " | ")

	if !strings.Contains(joined, "⚠️ /login needed") {
		t.Errorf("expected login needed, got %q", joined)
	}
}

func TestAppendUsageSegmentsSuccess(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	resetsAt := time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)

	keychain.GetFn = func() (string, error) { return "test-token", nil }
	usage.HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return []byte(`{
			"five_hour": {"utilization": 30, "resets_at": "` + resetsAt + `"},
			"seven_day": {"utilization": 45, "resets_at": "` + resetsAt + `"},
			"extra_usage": {"is_enabled": true, "monthly_limit": 5000, "used_credits": 128}
		}`), nil
	}

	segments := appendUsageSegments(nil)
	joined := strings.Join(segments, " | ")

	if !strings.Contains(joined, "7d: 45%") {
		t.Errorf("expected 7d quota, got %q", joined)
	}

	if !strings.Contains(joined, "5h: 30%") {
		t.Errorf("expected 5h quota, got %q", joined)
	}

	if !strings.Contains(joined, "💳 $128/$5000") {
		t.Errorf("expected extra usage, got %q", joined)
	}
}
