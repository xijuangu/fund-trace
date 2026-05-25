package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ValidConfig_LegacyFunds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
funds:
  - code: "011513"
  - code: "011925"
settings:
  refresh_interval_sec: 60
  alert_cooldown_min: 30
  max_concurrent_requests: 5
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(cfg.Assets))
	}
	if cfg.Assets[0].Code != "011513" {
		t.Errorf("expected 011513, got %s", cfg.Assets[0].Code)
	}
	if cfg.Assets[0].Kind != "fund" {
		t.Errorf("expected kind fund, got %s", cfg.Assets[0].Kind)
	}
	if cfg.Settings.RefreshIntervalSec != 60 {
		t.Errorf("expected 60, got %d", cfg.Settings.RefreshIntervalSec)
	}
	// Verify FundCodes works on legacy-loaded config.
	codes := cfg.FundCodes()
	if len(codes) != 2 {
		t.Errorf("expected 2 fund codes, got %d", len(codes))
	}
	if !cfg.loadedFromLegacy {
		t.Error("expected loadedFromLegacy to be true")
	}
}

func TestLoad_ValidConfig_NewAssets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
assets:
  - kind: fund
    code: "011513"
  - kind: stock
    market: sh
    code: "600519"
settings:
  refresh_interval_sec: 60
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(cfg.Assets))
	}
	if cfg.Assets[1].Kind != "stock" {
		t.Errorf("expected kind stock, got %s", cfg.Assets[1].Kind)
	}
	if cfg.Assets[1].Market != "sh" {
		t.Errorf("expected market sh, got %s", cfg.Assets[1].Market)
	}
	if cfg.Assets[1].Code != "600519" {
		t.Errorf("expected code 600519, got %s", cfg.Assets[1].Code)
	}
	// Verify StockEntries works.
	stocks := cfg.StockEntries()
	if len(stocks) != 1 {
		t.Errorf("expected 1 stock, got %d", len(stocks))
	}
	if stocks[0].Market != "sh" || stocks[0].Code != "600519" {
		t.Errorf("expected sh:600519, got %s:%s", stocks[0].Market, stocks[0].Code)
	}
	codes := cfg.FundCodes()
	if len(codes) != 1 {
		t.Errorf("expected 1 fund code, got %d", len(codes))
	}
}

func TestLoad_AssetsBothFundsAndAssets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
funds:
  - code: "000001"
assets:
  - kind: fund
    code: "011513"
  - kind: stock
    market: sz
    code: "000001"
settings:
  refresh_interval_sec: 60
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	// assets: takes priority when both present.
	if len(cfg.Assets) != 2 {
		t.Errorf("expected 2 assets from assets: key, got %d", len(cfg.Assets))
	}
	if cfg.Assets[0].Code != "011513" {
		t.Errorf("expected first asset 011513, got %s", cfg.Assets[0].Code)
	}
	if cfg.Assets[1].Kind != "stock" || cfg.Assets[1].Code != "000001" {
		t.Errorf("expected second asset stock 000001, got kind=%s code=%s", cfg.Assets[1].Kind, cfg.Assets[1].Code)
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

func TestLoad_EmptyAssets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("assets: []\nsettings:\n  refresh_interval_sec: 60\n"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty assets")
	}
}

func TestValidate_InvalidCode(t *testing.T) {
	cfg := &Config{
		Assets: []AssetEntry{{Kind: "fund", Code: "123"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid fund code")
	}
}

func TestValidate_StockWithInvalidMarket(t *testing.T) {
	cfg := &Config{
		Assets: []AssetEntry{{Kind: "stock", Market: "jp", Code: "00001"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for unknown market")
	}
}

func TestValidate_StockWithAutoInferMarket(t *testing.T) {
	// 600xxx → sh
	cfg := &Config{
		Assets:   []AssetEntry{{Kind: "stock", Code: "600519"}},
		Settings: Settings{RefreshIntervalSec: 60},
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Assets[0].Market != "sh" {
		t.Errorf("expected market sh, got %s", cfg.Assets[0].Market)
	}
}

func TestValidate_StockBeijingExchange(t *testing.T) {
	cfg := &Config{
		Assets: []AssetEntry{{Kind: "stock", Code: "430001"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for Beijing stock exchange code")
	}
}

func TestValidate_FillsDefaults(t *testing.T) {
	cfg := &Config{
		Assets:   []AssetEntry{{Kind: "fund", Code: "011513"}},
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
	if s.ChangeColorScheme != "green_up_red_down" {
		t.Errorf("expected default color scheme green_up_red_down, got %q", s.ChangeColorScheme)
	}
}

func TestResolveDBPath_DefaultConfigUsesCurrentDirectory(t *testing.T) {
	got := ResolveDBPath("config.yaml", "")
	if got != "fund-trace.db" {
		t.Fatalf("expected fund-trace.db, got %s", got)
	}
}

func TestResolveDBPath_CustomConfigUsesConfigDirectory(t *testing.T) {
	got := ResolveDBPath(filepath.Join("/tmp", "fund-trace-review", "config.yaml"), "")
	want := filepath.Join("/tmp", "fund-trace-review", "fund-trace.db")
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestResolveDBPath_ExplicitRelativePathUsesConfigDirectory(t *testing.T) {
	got := ResolveDBPath(filepath.Join("/tmp", "fund-trace-review", "config.yaml"), "data/app.db")
	want := filepath.Join("/tmp", "fund-trace-review", "data", "app.db")
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestResolveDBPath_ExplicitAbsolutePathWins(t *testing.T) {
	got := ResolveDBPath(filepath.Join("/tmp", "fund-trace-review", "config.yaml"), "/var/tmp/custom.db")
	if got != "/var/tmp/custom.db" {
		t.Fatalf("expected explicit absolute path, got %s", got)
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
	if len(cfg.Assets) != 13 {
		t.Errorf("expected 13 default assets, got %d", len(cfg.Assets))
	}
	for _, a := range cfg.Assets {
		if a.Kind != "fund" {
			t.Errorf("expected default asset kind fund, got %s", a.Kind)
		}
	}
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
	if len(cfg.Assets) != 1 {
		t.Errorf("expected 1 asset, got %d", len(cfg.Assets))
	}
	if cfg.Assets[0].Code != "000001" {
		t.Errorf("expected 000001, got %s", cfg.Assets[0].Code)
	}
	if cfg.Settings.RefreshIntervalSec != 120 {
		t.Errorf("expected 120, got %d", cfg.Settings.RefreshIntervalSec)
	}
}

func TestAddRemoveAsset(t *testing.T) {
	cfg := &Config{
		Assets: []AssetEntry{
			{Kind: "fund", Code: "011513"},
		},
	}
	cfg.AddFund("011925")
	if len(cfg.Assets) != 2 {
		t.Errorf("expected 2 after AddFund, got %d", len(cfg.Assets))
	}
	cfg.AddStock("sh", "600519")
	if len(cfg.Assets) != 3 {
		t.Errorf("expected 3 after AddStock, got %d", len(cfg.Assets))
	}
	cfg.RemoveAsset("fund", "", "011513")
	if len(cfg.Assets) != 2 {
		t.Errorf("expected 2 after RemoveAsset, got %d", len(cfg.Assets))
	}
	cfg.RemoveAsset("stock", "sh", "600519")
	if len(cfg.Assets) != 1 {
		t.Errorf("expected 1 after RemoveAsset, got %d", len(cfg.Assets))
	}
	if cfg.Assets[0].Code != "011925" {
		t.Errorf("expected remaining asset 011925, got %s", cfg.Assets[0].Code)
	}
}

func TestSave_WritesAssetsFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Assets: []AssetEntry{
			{Kind: "fund", Code: "011513"},
			{Kind: "stock", Market: "sh", Code: "600519"},
		},
		Settings: DefaultSettings(),
	}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "assets:") {
		t.Errorf("saved config missing 'assets:' key")
	}
	if strings.Contains(content, "funds:") {
		t.Errorf("saved config should NOT contain legacy 'funds:' key")
	}
}
