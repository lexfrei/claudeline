package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lexfrei/claudeline/internal/config"
	"github.com/lexfrei/claudeline/internal/keychain"
	"github.com/lexfrei/claudeline/internal/status"
	"github.com/lexfrei/claudeline/internal/usage"
)

const testToken = "test-token"

func defaultCfg() *config.Config {
	cfg := config.Defaults()

	return &cfg
}

func setupTestEnv(t *testing.T) func() {
	t.Helper()

	dir := t.TempDir()

	origStatusPath := status.CachePath
	origUsagePath := usage.CachePath
	origLastGoodPath := usage.LastGoodCachePath
	origStatusHTTP := status.HTTPGetFn
	origUsageHTTP := usage.HTTPGetFn
	origToken := keychain.GetFn
	origStatusTTL := status.CacheTTL
	origUsageTTL := usage.CacheTTL

	status.CachePath = filepath.Join(dir, "status-cache.json")
	usage.CachePath = filepath.Join(dir, "usage-cache.json")
	usage.LastGoodCachePath = filepath.Join(dir, "usage-last-good.json")

	return func() {
		status.CachePath = origStatusPath
		usage.CachePath = origUsagePath
		usage.LastGoodCachePath = origLastGoodPath
		status.HTTPGetFn = origStatusHTTP
		usage.HTTPGetFn = origUsageHTTP
		keychain.GetFn = origToken
		status.CacheTTL = origStatusTTL
		usage.CacheTTL = origUsageTTL
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

	got := buildStatusline([]byte(`{}`), defaultCfg())

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
	got := buildStatusline([]byte(input), defaultCfg())

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
	got := buildStatusline([]byte(input), defaultCfg())

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
	got := buildStatusline([]byte(input), defaultCfg())

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

	got := buildStatusline([]byte(`{}`), defaultCfg())

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

	got := buildStatusline([]byte(`not json`), defaultCfg())

	if !strings.Contains(got, "🤖 Claude") {
		t.Errorf("expected graceful degradation, got %q", got)
	}
}

func TestAppendUsageSegmentsLoginNeeded(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return testToken, nil }
	usage.HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return []byte(`{"error":{"type":"authentication_error"}}`), nil
	}

	segments := appendUsageSegments(nil, defaultCfg())
	joined := strings.Join(segments, " | ")

	if !strings.Contains(joined, "⚠️ /login needed") {
		t.Errorf("expected login needed, got %q", joined)
	}
}

func TestAppendUsageSegmentsRateLimited(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	resetsAt := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)

	err := os.WriteFile(usage.LastGoodCachePath, []byte(`{
		"five_hour": {"utilization": 42, "resets_at": "`+resetsAt+`"},
		"seven_day": {"utilization": 99, "resets_at": "`+resetsAt+`"}
	}`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	keychain.GetFn = func() (string, error) { return testToken, nil }
	usage.HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return []byte(`{"error":{"type":"rate_limit_error"}}`), nil
	}

	segments := appendUsageSegments(nil, defaultCfg())
	joined := strings.Join(segments, " | ")

	if strings.Contains(joined, "/login needed") {
		t.Errorf("rate_limit_error should not show login needed, got %q", joined)
	}

	if !strings.Contains(joined, "⛔ limit hit") {
		t.Errorf("expected explicit rate-limit segment, got %q", joined)
	}

	if !strings.Contains(joined, "7d: ?%") {
		t.Errorf("expected stale quota segments, got %q", joined)
	}
}

func TestAppendUsageSegmentsSuccess(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	resetsAt := time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)

	keychain.GetFn = func() (string, error) { return testToken, nil }
	usage.HTTPGetFn = func(_ string, _ map[string]string, _ time.Duration) ([]byte, error) {
		return []byte(`{
			"five_hour": {"utilization": 30, "resets_at": "` + resetsAt + `"},
			"seven_day": {"utilization": 45, "resets_at": "` + resetsAt + `"},
			"extra_usage": {"is_enabled": true, "monthly_limit": 5000, "used_credits": 128}
		}`), nil
	}

	segments := appendUsageSegments(nil, defaultCfg())
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

func TestBuildStatuslineNoModel(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = failHTTP
	usage.HTTPGetFn = failHTTP

	cfg := defaultCfg()
	cfg.Segments.Model = false

	got := buildStatusline([]byte(`{}`), cfg)

	if strings.Contains(got, "🤖") {
		t.Errorf("expected no model segment, got %q", got)
	}
}

func TestBuildStatuslineNoCost(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = failHTTP
	usage.HTTPGetFn = failHTTP

	cfg := defaultCfg()
	cfg.Segments.Cost = false

	got := buildStatusline([]byte(`{}`), cfg)

	if strings.Contains(got, "💰") {
		t.Errorf("expected no cost segment, got %q", got)
	}
}

func TestBuildStatuslineNoQuota(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = failHTTP
	usage.HTTPGetFn = failHTTP

	cfg := defaultCfg()
	cfg.Segments.Quota = false
	cfg.Segments.Credits = false

	got := buildStatusline([]byte(`{}`), cfg)

	if strings.Contains(got, "⏳") || strings.Contains(got, "7d") {
		t.Errorf("expected no quota segments, got %q", got)
	}
}

func TestBuildStatuslineAllDisabled(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = failHTTP
	usage.HTTPGetFn = failHTTP

	cfg := defaultCfg()
	cfg.Segments.Model = false
	cfg.Segments.Cost = false
	cfg.Segments.Status = false
	cfg.Segments.Context = false
	cfg.Segments.Compactions = false
	cfg.Segments.Quota = false
	cfg.Segments.Credits = false

	got := buildStatusline([]byte(`{}`), cfg)

	if got != "" {
		t.Errorf("expected empty output, got %q", got)
	}
}

func TestNewRootCmdVersion(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewRootCmdWithFlags(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = failHTTP
	usage.HTTPGetFn = failHTTP

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--no-model", "--no-cost", "--config", "/nonexistent/config.toml"})
	cmd.SetIn(strings.NewReader(`{}`))

	err := cmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewRootCmdWithConfigFile(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	keychain.GetFn = func() (string, error) { return "", keychain.ErrNoToken }
	status.HTTPGetFn = failHTTP
	usage.HTTPGetFn = failHTTP

	configContent := `
[segments]
model = false

[cache]
usage_ttl = "30s"
`

	configPath := filepath.Join(t.TempDir(), "config.toml")

	writeErr := os.WriteFile(configPath, []byte(configContent), 0o600)
	if writeErr != nil {
		t.Fatal(writeErr)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--config", configPath})
	cmd.SetIn(strings.NewReader(`{}`))

	err := cmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if usage.CacheTTL != 30*time.Second {
		t.Errorf("expected usage TTL 30s from config, got %v", usage.CacheTTL)
	}
}

func TestApplyFlagOverrides(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--no-model", "--no-quota", "--no-credits"})

	parseErr := cmd.ParseFlags([]string{"--no-model", "--no-quota", "--no-credits"})
	if parseErr != nil {
		t.Fatal(parseErr)
	}

	cfg := config.Defaults()
	applyFlagOverrides(cmd, &cfg)

	if cfg.Segments.Model {
		t.Error("expected model disabled by flag")
	}

	if cfg.Segments.Quota {
		t.Error("expected quota disabled by flag")
	}

	if cfg.Segments.Credits {
		t.Error("expected credits disabled by flag")
	}

	if !cfg.Segments.Cost {
		t.Error("expected cost still enabled")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	t.Parallel()

	got := defaultConfigPath()
	if got == "" {
		t.Skip("could not determine home directory")
	}

	if !strings.HasSuffix(got, ".claudelinerc.toml") {
		t.Errorf("expected path ending with .claudelinerc.toml, got %q", got)
	}
}

func TestFlagSetUnknownFlag(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()

	if flagSet(cmd, "nonexistent-flag") {
		t.Error("expected false for unknown flag")
	}
}
