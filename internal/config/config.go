package config

import (
	"errors"
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

// DefaultFunds returns a starter set of common index-tracking funds.
func DefaultFunds() []FundEntry {
	return []FundEntry{
		{Code: "011513"}, // 天弘中证新能源车C
		{Code: "011925"}, // 嘉实港股互联网产业核心资产C
		{Code: "017435"}, // 华宝中证沪港深新消费指数C
		{Code: "012734"}, // 易方达中证人工智能主题ETF联接C
		{Code: "008087"}, // 华夏中证5G通信主题ETF联接C
		{Code: "011609"}, // 易方达上证科创50联接C
		{Code: "012349"}, // 天弘恒生科技ETF联接C
		{Code: "007531"}, // 华宝券商ETF联接C
		{Code: "001595"}, // 天弘中证银行ETF联接C
		{Code: "016068"}, // 鹏华新能源汽车混合C
		{Code: "021492"}, // 中航远见领航混合发起C
		{Code: "562500"}, // 机器人ETF华夏
		{Code: "024913"}, // 华夏国证通用航空产业ETF发起式联接C
	}
}

func defaultConfig() Config {
	return Config{
		Funds:    DefaultFunds(),
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
