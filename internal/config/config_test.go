package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	t.Parallel()

	cfg := Defaults()

	if !cfg.Segments.Model {
		t.Error("expected model segment enabled by default")
	}

	if !cfg.Segments.Cost {
		t.Error("expected cost segment enabled by default")
	}

	if !cfg.Segments.Status {
		t.Error("expected status segment enabled by default")
	}

	if !cfg.Segments.Context {
		t.Error("expected context segment enabled by default")
	}

	if !cfg.Segments.Compactions {
		t.Error("expected compactions segment enabled by default")
	}

	if !cfg.Segments.Quota {
		t.Error("expected quota segment enabled by default")
	}

	if !cfg.Segments.Credits {
		t.Error("expected credits segment enabled by default")
	}

	if cfg.Cache.UsageTTL != 60*time.Second {
		t.Errorf("expected usage TTL 60s, got %v", cfg.Cache.UsageTTL)
	}

	if cfg.Cache.StatusTTL != 15*time.Second {
		t.Errorf("expected status TTL 15s, got %v", cfg.Cache.StatusTTL)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()

	cfg := Load("/nonexistent/config.toml")

	if !cfg.Segments.Model || !cfg.Segments.Cost {
		t.Error("expected defaults when config file is missing")
	}
}

func TestLoadEmptyPath(t *testing.T) {
	t.Parallel()

	cfg := Load("")

	if !cfg.Segments.Quota {
		t.Error("expected defaults when config path is empty")
	}
}

func TestLoadPartialConfig(t *testing.T) {
	t.Parallel()

	content := `
[segments]
cost = false
status = false
`

	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Load(configPath)

	if !cfg.Segments.Model {
		t.Error("expected model segment enabled (not in config)")
	}

	if cfg.Segments.Cost {
		t.Error("expected cost segment disabled")
	}

	if cfg.Segments.Status {
		t.Error("expected status segment disabled")
	}

	if !cfg.Segments.Quota {
		t.Error("expected quota segment enabled (not in config)")
	}

	if cfg.Cache.UsageTTL != 60*time.Second {
		t.Errorf("expected default usage TTL, got %v", cfg.Cache.UsageTTL)
	}
}

func TestLoadFullConfig(t *testing.T) {
	t.Parallel()

	content := `
[segments]
model = false
cost = false
status = false
context = false
compactions = false
quota = false
credits = false

[cache]
usage_ttl = "120s"
status_ttl = "30s"
`

	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Load(configPath)

	if cfg.Segments.Model || cfg.Segments.Cost || cfg.Segments.Status ||
		cfg.Segments.Context || cfg.Segments.Compactions || cfg.Segments.Quota || cfg.Segments.Credits {
		t.Error("expected all segments disabled")
	}

	if cfg.Cache.UsageTTL != 120*time.Second {
		t.Errorf("expected usage TTL 120s, got %v", cfg.Cache.UsageTTL)
	}

	if cfg.Cache.StatusTTL != 30*time.Second {
		t.Errorf("expected status TTL 30s, got %v", cfg.Cache.StatusTTL)
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("not valid toml [[["), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Load(configPath)

	if !cfg.Segments.Model {
		t.Error("expected defaults on invalid TOML")
	}
}
