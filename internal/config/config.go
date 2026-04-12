// Package config provides configuration loading from TOML files and CLI flags.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultUsageTTL  = 10 * time.Minute
	defaultStatusTTL = 15 * time.Second
)

// Cost mode values.
const (
	CostAuto = "auto"
	CostOn   = "true"
	CostOff  = "false"
)

// NormalizeCostMode converts various user inputs to a canonical cost mode.
// Accepts: "auto", "true"/"1"/"on", "false"/"0"/"off". Returns "" for unknown values.
func NormalizeCostMode(raw string) string {
	switch raw {
	case CostAuto, "":
		return CostAuto
	case "true", "1", "on":
		return CostOn
	case "false", "0", "off":
		return CostOff
	default:
		return ""
	}
}

// Segments controls which statusline segments are displayed.
type Segments struct {
	Model         bool   `mapstructure:"model"`
	Worktree      bool   `mapstructure:"worktree"`
	Cost          string `mapstructure:"cost"`
	Status        bool   `mapstructure:"status"`
	Context       bool   `mapstructure:"context"`
	Compactions   bool   `mapstructure:"compactions"`
	Quota         bool   `mapstructure:"quota"`
	PerModelQuota bool   `mapstructure:"per_model_quota"`
	Credits       bool   `mapstructure:"credits"`
	OffPeak       bool   `mapstructure:"offpeak"`
}

// Cache controls cache TTL durations.
type Cache struct {
	UsageTTL  time.Duration `mapstructure:"usage_ttl"`
	StatusTTL time.Duration `mapstructure:"status_ttl"`
}

// Config holds all claudeline configuration.
type Config struct {
	Segments    Segments `mapstructure:"segments"`
	Cache       Cache    `mapstructure:"cache"`
	MacInsecure bool     `mapstructure:"mac_insecure"`
}

// Defaults returns a Config with all segments enabled and default TTLs.
func Defaults() Config {
	return Config{
		Segments: Segments{
			Model:       true,
			Worktree:    true,
			Cost:        CostAuto,
			Status:      true,
			Context:     true,
			Compactions: true,
			Quota:       true,
			Credits:     true,
			OffPeak:     true,
		},
		Cache: Cache{
			UsageTTL:  defaultUsageTTL,
			StatusTTL: defaultStatusTTL,
		},
	}
}

// Load reads configuration from a TOML file at the given path.
// Missing file or parse errors result in defaults being used.
func Load(configPath string) Config {
	cfg := Defaults()

	if configPath == "" {
		return cfg
	}

	viperInstance := viper.New()
	viperInstance.SetConfigFile(configPath)
	viperInstance.SetConfigType("toml")

	setViperDefaults(viperInstance)

	readErr := viperInstance.ReadInConfig()
	if readErr != nil {
		return cfg
	}

	unmarshalErr := viperInstance.Unmarshal(&cfg)
	if unmarshalErr != nil {
		fmt.Fprintf(os.Stderr, "claudeline: config parse error: %v\n", unmarshalErr)

		return Defaults()
	}

	cfg.Segments.Cost = NormalizeCostMode(cfg.Segments.Cost)
	if cfg.Segments.Cost == "" {
		fmt.Fprintf(os.Stderr, "claudeline: invalid cost mode %q, using auto\n", viperInstance.GetString("segments.cost"))

		cfg.Segments.Cost = CostAuto
	}

	return cfg
}

// knownKeys is the set of all valid configuration keys.
var knownKeys = map[string]bool{
	"segments.model":           true,
	"segments.worktree":        true,
	"segments.cost":            true,
	"segments.status":          true,
	"segments.context":         true,
	"segments.compactions":     true,
	"segments.quota":           true,
	"segments.per_model_quota": true,
	"segments.credits":         true,
	"segments.offpeak":         true,
	"cache.usage_ttl":          true,
	"cache.status_ttl":         true,
	"mac_insecure":             true,
}

// Validate checks the config file at the given path for errors.
// Returns a list of human-readable problems. Empty list means valid.
func Validate(configPath string) []string {
	if configPath == "" {
		return nil
	}

	viperInstance := viper.New()
	viperInstance.SetConfigFile(configPath)
	viperInstance.SetConfigType("toml")

	setViperDefaults(viperInstance)

	readErr := viperInstance.ReadInConfig()
	if readErr != nil {
		return []string{fmt.Sprintf("cannot read config: %v", readErr)}
	}

	var problems []string

	// Check for unknown keys (typos).
	for _, key := range viperInstance.AllKeys() {
		if !knownKeys[key] {
			problems = append(problems, fmt.Sprintf("unknown key %q (typo?)", key))
		}
	}

	var cfg Config

	unmarshalErr := viperInstance.Unmarshal(&cfg)
	if unmarshalErr != nil {
		return append(problems, fmt.Sprintf("cannot parse config: %v", unmarshalErr))
	}

	problems = append(problems, validateSegments(&cfg.Segments, viperInstance)...)
	problems = append(problems, validateCache(&cfg.Cache)...)

	return problems
}

func validateSegments(seg *Segments, v *viper.Viper) []string {
	var problems []string

	if raw := seg.Cost; NormalizeCostMode(raw) == "" {
		problems = append(problems, fmt.Sprintf("segments.cost: unknown value %q (expected auto, true, or false)", raw))
	}

	boolFields := []struct {
		key string
		raw any
	}{
		{"segments.model", v.Get("segments.model")},
		{"segments.worktree", v.Get("segments.worktree")},
		{"segments.status", v.Get("segments.status")},
		{"segments.context", v.Get("segments.context")},
		{"segments.compactions", v.Get("segments.compactions")},
		{"segments.quota", v.Get("segments.quota")},
		{"segments.per_model_quota", v.Get("segments.per_model_quota")},
		{"segments.credits", v.Get("segments.credits")},
		{"segments.offpeak", v.Get("segments.offpeak")},
	}

	for _, field := range boolFields {
		if field.raw == nil {
			continue
		}

		if _, ok := field.raw.(bool); !ok {
			problems = append(problems, fmt.Sprintf("%s: expected true or false, got %v", field.key, field.raw))
		}
	}

	if v.Get("mac_insecure") != nil {
		if _, ok := v.Get("mac_insecure").(bool); !ok {
			problems = append(problems, fmt.Sprintf("mac_insecure: expected true or false, got %v", v.Get("mac_insecure")))
		}
	}

	return problems
}

func validateCache(cache *Cache) []string {
	var problems []string

	if cache.UsageTTL < 0 {
		problems = append(problems, fmt.Sprintf("cache.usage_ttl: negative duration %v", cache.UsageTTL))
	}

	if cache.StatusTTL < 0 {
		problems = append(problems, fmt.Sprintf("cache.status_ttl: negative duration %v", cache.StatusTTL))
	}

	return problems
}

func setViperDefaults(viperInstance *viper.Viper) {
	viperInstance.SetDefault("segments.model", true)
	viperInstance.SetDefault("segments.worktree", true)
	viperInstance.SetDefault("segments.cost", CostAuto)
	viperInstance.SetDefault("segments.status", true)
	viperInstance.SetDefault("segments.context", true)
	viperInstance.SetDefault("segments.compactions", true)
	viperInstance.SetDefault("segments.quota", true)
	viperInstance.SetDefault("segments.per_model_quota", false)
	viperInstance.SetDefault("segments.credits", true)
	viperInstance.SetDefault("segments.offpeak", true)
	viperInstance.SetDefault("mac_insecure", false)
	viperInstance.SetDefault("cache.usage_ttl", defaultUsageTTL)
	viperInstance.SetDefault("cache.status_ttl", defaultStatusTTL)
}
