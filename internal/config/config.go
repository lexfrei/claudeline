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

func setViperDefaults(viperInstance *viper.Viper) {
	viperInstance.SetDefault("segments.model", true)
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
