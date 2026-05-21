package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
funds:
  - code: "011513"
  - code: "011925"
settings:
  refresh_interval_sec: 60
  cache_ttl_min: 6
  alert_cooldown_min: 30
  max_concurrent_requests: 5
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Funds) != 2 {
		t.Errorf("expected 2 funds, got %d", len(cfg.Funds))
	}
	if cfg.Funds[0].Code != "011513" {
		t.Errorf("expected 011513, got %s", cfg.Funds[0].Code)
	}
	if cfg.Settings.RefreshIntervalSec != 60 {
		t.Errorf("expected 60, got %d", cfg.Settings.RefreshIntervalSec)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_MalformedYaml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("funds: [unclosed"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed yaml")
	}
}

func TestLoad_EmptyFunds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("funds: []\nsettings:\n  refresh_interval_sec: 60\n"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty funds")
	}
}

func TestValidate_InvalidCode(t *testing.T) {
	cfg := &Config{
		Funds: []FundEntry{{Code: "123"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid fund code")
	}
}

func TestValidate_FillsDefaults(t *testing.T) {
	cfg := &Config{
		Funds:    []FundEntry{{Code: "011513"}},
		Settings: Settings{},
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Settings.RefreshIntervalSec != 60 {
		t.Errorf("expected default 60, got %d", cfg.Settings.RefreshIntervalSec)
	}
	if cfg.Settings.MaxConcurrentRequests != 5 {
		t.Errorf("expected default 5, got %d", cfg.Settings.MaxConcurrentRequests)
	}
}

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	if s.RefreshIntervalSec != 60 {
		t.Errorf("expected 60, got %d", s.RefreshIntervalSec)
	}
}

func TestDefaultFunds(t *testing.T) {
	funds := DefaultFunds()
	if len(funds) != 13 {
		t.Errorf("expected 13 default funds, got %d", len(funds))
	}
	for _, f := range funds {
		if len(f.Code) != 6 {
			t.Errorf("fund code %q is not 6 digits", f.Code)
		}
	}
}

func TestLoadOrCreate_GeneratesOnMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("LoadOrCreate() error: %v", err)
	}
	if len(cfg.Funds) != 13 {
		t.Errorf("expected 13 default funds, got %d", len(cfg.Funds))
	}
	// Verify file was written
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("config.yaml was not created on disk")
	}
}

func TestLoadOrCreate_UsesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
funds:
  - code: "000001"
settings:
  refresh_interval_sec: 120
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("LoadOrCreate() error: %v", err)
	}
	if len(cfg.Funds) != 1 {
		t.Errorf("expected 1 fund, got %d", len(cfg.Funds))
	}
	if cfg.Funds[0].Code != "000001" {
		t.Errorf("expected 000001, got %s", cfg.Funds[0].Code)
	}
	if cfg.Settings.RefreshIntervalSec != 120 {
		t.Errorf("expected 120, got %d", cfg.Settings.RefreshIntervalSec)
	}
}
