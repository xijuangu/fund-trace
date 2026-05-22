package config

import (
	"errors"
	"fmt"
	"os"

	"fund-trace/internal/model"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Funds is the legacy list (kept for backward-compat reading).
	Funds []FundEntry `yaml:"funds,omitempty"`
	// Assets is the recommended list. When both are present, Assets takes priority.
	Assets   []AssetEntry `yaml:"assets,omitempty"`
	Settings Settings     `yaml:"settings"`

	// runtime: set after load to indicate source was legacy funds:
	loadedFromLegacy bool
}

// FundEntry is the legacy per-fund entry (code only).
type FundEntry struct {
	Code string `yaml:"code"`
}

// AssetEntry is the new per-asset entry.
type AssetEntry struct {
	Kind   string `yaml:"kind"`   // "fund" or "stock"
	Market string `yaml:"market"` // "" for fund, "sh"/"sz" for stock
	Code   string `yaml:"code"`
}

type Settings struct {
	RefreshIntervalSec    int `yaml:"refresh_interval_sec"`
	CacheTTLMin           int `yaml:"cache_ttl_min"`
	AlertCooldownMin      int `yaml:"alert_cooldown_min"`
	MaxConcurrentRequests int `yaml:"max_concurrent_requests"`
}

// FundCodes extracts fund codes from the configuration.
func (c *Config) FundCodes() []string {
	codes := make([]string, 0)
	for _, a := range c.Assets {
		if a.Kind == "fund" || (a.Kind == "" && a.Market == "") {
			codes = append(codes, a.Code)
		}
	}
	return codes
}

// StockCodes extracts stock (market, code) pairs from the configuration.
// Returns nil if no stocks are configured.
func (c *Config) StockEntries() []struct{ Market, Code string } {
	var entries []struct{ Market, Code string }
	for _, a := range c.Assets {
		if a.Kind == "stock" {
			entries = append(entries, struct{ Market, Code string }{a.Market, a.Code})
		}
	}
	return entries
}

// AddFund appends a fund to the configuration.
func (c *Config) AddFund(code string) {
	c.Assets = append(c.Assets, AssetEntry{Kind: "fund", Code: code})
}

// AddStock appends a stock to the configuration.
func (c *Config) AddStock(market, code string) {
	c.Assets = append(c.Assets, AssetEntry{Kind: "stock", Market: market, Code: code})
}

// RemoveAsset removes an asset by kind, market, and code.
func (c *Config) RemoveAsset(kind, market, code string) {
	filtered := make([]AssetEntry, 0, len(c.Assets))
	for _, a := range c.Assets {
		if a.Kind == kind && a.Market == market && a.Code == code {
			continue
		}
		filtered = append(filtered, a)
	}
	c.Assets = filtered
}

// AllAssetCodes returns all kind+market+code combinations for seeding the store.
func (c *Config) AllAssetCodes() (fundCodes []string, stocks []struct{ Market, Code string }) {
	fundCodes = c.FundCodes()
	stocks = c.StockEntries()
	return
}

func DefaultFunds() []FundEntry {
	return []FundEntry{
		{Code: "011513"},
		{Code: "011925"},
		{Code: "017435"},
		{Code: "012734"},
		{Code: "008087"},
		{Code: "011609"},
		{Code: "012349"},
		{Code: "007531"},
		{Code: "001595"},
		{Code: "016068"},
		{Code: "021492"},
		{Code: "562500"},
		{Code: "024913"},
	}
}

func defaultAssets() []AssetEntry {
	funds := DefaultFunds()
	assets := make([]AssetEntry, len(funds))
	for i, f := range funds {
		assets[i] = AssetEntry{Kind: "fund", Code: f.Code}
	}
	return assets
}

func defaultConfig() Config {
	return Config{
		Assets:   defaultAssets(),
		Settings: DefaultSettings(),
	}
}

// LoadOrCreate tries to load config from path. If the file does not exist,
// it generates a default one at that path and returns it.
func LoadOrCreate(path string) (*Config, error) {
	cfg, err := Load(path)
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	dc := defaultConfig()
	data, err := yaml.Marshal(&dc)
	if err != nil {
		return nil, fmt.Errorf("marshal default config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("write default config %s: %w", path, err)
	}
	fmt.Printf("默认配置文件已生成: %s\n", path)
	return Load(path)
}

// rawConfig is used for two-pass YAML parsing: detect which keys exist
// before deciding whether to use the legacy funds key.
type rawConfig struct {
	Funds    yaml.Node `yaml:"funds"`
	Assets   yaml.Node `yaml:"assets"`
	Settings Settings  `yaml:"settings"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	// First pass: detect whether assets key exists.
	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	hasFunds := raw.Funds.Kind != 0
	hasAssets := raw.Assets.Kind != 0

	// If neither key exists, try parsing as legacy with only funds possible.
	// This covers the case where someone is using the old format.
	var cfg Config

	if hasAssets {
		// New format: parse assets.
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing config file %s: %w", path, err)
		}
	} else if hasFunds {
		// Old format: parse funds then convert to assets.
		type legacyConfig struct {
			Funds    []FundEntry `yaml:"funds"`
			Settings Settings    `yaml:"settings"`
		}
		var lc legacyConfig
		if err := yaml.Unmarshal(data, &lc); err != nil {
			return nil, fmt.Errorf("parsing legacy config file %s: %w", path, err)
		}
		cfg.Settings = lc.Settings
		cfg.Assets = make([]AssetEntry, len(lc.Funds))
		for i, f := range lc.Funds {
			cfg.Assets[i] = AssetEntry{Kind: "fund", Code: f.Code}
		}
		cfg.loadedFromLegacy = true
	} else {
		return nil, fmt.Errorf("config: no assets or funds configured")
	}

	if len(cfg.Assets) == 0 {
		return nil, fmt.Errorf("config: no assets configured (assets list is empty)")
	}

	return &cfg, nil
}

func (c *Config) Save(path string) error {
	// Always save in new assets format.
	saveCfg := struct {
		Assets   []AssetEntry `yaml:"assets"`
		Settings Settings     `yaml:"settings"`
	}{
		Assets:   c.Assets,
		Settings: c.Settings,
	}
	data, err := yaml.Marshal(saveCfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

func (c *Config) Validate() error {
	if len(c.Assets) == 0 {
		return fmt.Errorf("no assets configured")
	}
	for i, a := range c.Assets {
		if a.Kind == "" && a.Market == "" {
			// Treat as fund (legacy or implicit).
			if len(a.Code) != 6 {
				return fmt.Errorf("asset[%d]: invalid fund code %q (must be 6 digits)", i, a.Code)
			}
		} else if a.Kind == "fund" {
			if len(a.Code) != 6 {
				return fmt.Errorf("asset[%d]: invalid fund code %q (must be 6 digits)", i, a.Code)
			}
		} else if a.Kind == "stock" {
			if len(a.Code) != 6 {
				return fmt.Errorf("asset[%d]: invalid stock code %q (must be 6 digits)", i, a.Code)
			}
			if a.Market == "" {
				// Try to infer.
				mkt, err := model.InferStockMarket(a.Code)
				if err != nil {
					return fmt.Errorf("asset[%d]: %w", i, err)
				}
				c.Assets[i].Market = mkt
			} else if a.Market != "sh" && a.Market != "sz" {
				return fmt.Errorf("asset[%d]: unknown market %q for stock (expected sh or sz)", i, a.Market)
			}
		} else {
			return fmt.Errorf("asset[%d]: unknown kind %q (expected fund or stock)", i, a.Kind)
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
