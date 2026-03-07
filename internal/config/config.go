// Package config provides configuration loading from TOML files and CLI flags.
package config

import (
	"time"

	"github.com/spf13/viper"
)

const (
	defaultUsageTTL  = 5 * time.Minute
	defaultStatusTTL = 15 * time.Second
)

// Segments controls which statusline segments are displayed.
type Segments struct {
	Model       bool `mapstructure:"model"`
	Cost        bool `mapstructure:"cost"`
	Status      bool `mapstructure:"status"`
	Context     bool `mapstructure:"context"`
	Compactions bool `mapstructure:"compactions"`
	Quota       bool `mapstructure:"quota"`
	Credits     bool `mapstructure:"credits"`
}

// Cache controls cache TTL durations.
type Cache struct {
	UsageTTL  time.Duration `mapstructure:"usage_ttl"`
	StatusTTL time.Duration `mapstructure:"status_ttl"`
}

// Config holds all claudeline configuration.
type Config struct {
	Segments Segments `mapstructure:"segments"`
	Cache    Cache    `mapstructure:"cache"`
}

// Defaults returns a Config with all segments enabled and default TTLs.
func Defaults() Config {
	return Config{
		Segments: Segments{
			Model:       true,
			Cost:        true,
			Status:      true,
			Context:     true,
			Compactions: true,
			Quota:       true,
			Credits:     true,
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

	_ = viperInstance.Unmarshal(&cfg)

	return cfg
}

func setViperDefaults(viperInstance *viper.Viper) {
	viperInstance.SetDefault("segments.model", true)
	viperInstance.SetDefault("segments.cost", true)
	viperInstance.SetDefault("segments.status", true)
	viperInstance.SetDefault("segments.context", true)
	viperInstance.SetDefault("segments.compactions", true)
	viperInstance.SetDefault("segments.quota", true)
	viperInstance.SetDefault("segments.credits", true)
	viperInstance.SetDefault("cache.usage_ttl", defaultUsageTTL)
	viperInstance.SetDefault("cache.status_ttl", defaultStatusTTL)
}
