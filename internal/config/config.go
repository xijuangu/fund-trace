package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Funds    []FundEntry `yaml:"funds"`
	Settings Settings    `yaml:"settings"`
}

type FundEntry struct {
	Code string `yaml:"code"`
}

type Settings struct {
	RefreshIntervalSec    int `yaml:"refresh_interval_sec"`
	CacheTTLMin           int `yaml:"cache_ttl_min"`
	AlertCooldownMin      int `yaml:"alert_cooldown_min"`
	MaxConcurrentRequests int `yaml:"max_concurrent_requests"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	if len(cfg.Funds) == 0 {
		return nil, fmt.Errorf("config: no funds configured (funds list is empty)")
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Funds) == 0 {
		return fmt.Errorf("no funds configured")
	}
	for i, f := range c.Funds {
		if len(f.Code) != 6 {
			return fmt.Errorf("fund[%d]: invalid fund code %q (must be 6 digits)", i, f.Code)
		}
	}
	if c.Settings.RefreshIntervalSec <= 0 {
		c.Settings.RefreshIntervalSec = 60
	}
	if c.Settings.CacheTTLMin <= 0 {
		c.Settings.CacheTTLMin = 6
	}
	if c.Settings.AlertCooldownMin <= 0 {
		c.Settings.AlertCooldownMin = 30
	}
	if c.Settings.MaxConcurrentRequests <= 0 {
		c.Settings.MaxConcurrentRequests = 5
	}
	return nil
}

func DefaultSettings() Settings {
	return Settings{
		RefreshIntervalSec:    60,
		CacheTTLMin:           6,
		AlertCooldownMin:      30,
		MaxConcurrentRequests: 5,
	}
}
