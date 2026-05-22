package cmd

import (
	"path/filepath"
	"testing"

	"fund-trace/internal/config"
	"fund-trace/internal/model"
	"fund-trace/internal/store"
)

func TestIsStockHistoryRequest_DoesNotTreatConfiguredFundAsStock(t *testing.T) {
	cfg := &config.Config{
		Assets: []config.AssetEntry{
			{Kind: "fund", Code: "011513"},
		},
	}

	if isStockHistoryRequest(cfg, "011513") {
		t.Fatal("configured fund 011513 must not be treated as stock history")
	}
}

func TestPersistAddedFundSyncsAssetsAndConfig(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{Settings: config.DefaultSettings()}
	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := persistAddedFund(st, cfg, path, "011513", "测试基金", model.FundIndex); err != nil {
		t.Fatalf("persistAddedFund: %v", err)
	}

	assets, err := st.ListAssets()
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != 1 || assets[0].Kind != model.AssetKindFund || assets[0].Code != "011513" {
		t.Fatalf("expected one fund asset 011513, got %#v", assets)
	}
	if len(cfg.Assets) != 1 || cfg.Assets[0].Kind != "fund" || cfg.Assets[0].Code != "011513" {
		t.Fatalf("config assets not updated: %#v", cfg.Assets)
	}
}

func TestSeedConfiguredFundSyncsLegacyFundAndAssetTables(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := seedConfiguredFund(st, "011513"); err != nil {
		t.Fatalf("seedConfiguredFund: %v", err)
	}

	funds, err := st.ListFunds()
	if err != nil {
		t.Fatalf("list funds: %v", err)
	}
	if len(funds) != 1 || funds[0].Code != "011513" {
		t.Fatalf("expected legacy funds table to contain 011513, got %#v", funds)
	}

	assets, err := st.ListAssets()
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != 1 || assets[0].Kind != model.AssetKindFund || assets[0].Code != "011513" {
		t.Fatalf("expected assets table to contain fund 011513, got %#v", assets)
	}
}

func TestPersistRemovedFundSyncsAssetsAndConfig(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{
		Assets:   []config.AssetEntry{{Kind: "fund", Code: "011513"}},
		Settings: config.DefaultSettings(),
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := persistAddedFund(st, cfg, path, "011513", "测试基金", model.FundIndex); err != nil {
		t.Fatalf("persistAddedFund: %v", err)
	}

	if err := persistRemovedFund(st, cfg, path, "011513"); err != nil {
		t.Fatalf("persistRemovedFund: %v", err)
	}

	assets, err := st.ListAssets()
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != 0 {
		t.Fatalf("expected no assets after remove, got %#v", assets)
	}
	if len(cfg.Assets) != 0 {
		t.Fatalf("config assets not removed: %#v", cfg.Assets)
	}
}
