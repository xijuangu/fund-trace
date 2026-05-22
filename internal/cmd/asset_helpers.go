package cmd

import (
	"fund-trace/internal/config"
	"fund-trace/internal/model"
	"fund-trace/internal/store"
)

func isStockHistoryRequest(cfg *config.Config, code string) bool {
	isStock, _, _ := resolveStockHistoryRequest(cfg, code)
	return isStock
}

func resolveStockHistoryRequest(cfg *config.Config, code string) (isStock bool, market string, resolvedCode string) {
	if cfg == nil {
		return false, "", code
	}
	for _, a := range cfg.Assets {
		if a.Code != code {
			continue
		}
		switch a.Kind {
		case "fund", "":
			return false, "", code
		case "stock":
			return true, a.Market, a.Code
		}
	}
	return false, "", code
}

func seedConfiguredFund(st *store.Store, code string) error {
	if err := st.AddFund(code); err != nil {
		return err
	}
	return st.AddAssetSimple(model.AssetKindFund, "", code)
}

func persistAddedFund(st *store.Store, cfg *config.Config, configPath, code, name string, fundType model.FundType) error {
	if err := st.AddFundWithName(code, name, fundType); err != nil {
		return err
	}
	if err := st.AddAssetWithName(model.AssetKindFund, "", code, name, int(fundType)); err != nil {
		return err
	}
	cfg.AddFund(code)
	return cfg.Save(configPath)
}

func persistRemovedFund(st *store.Store, cfg *config.Config, configPath, code string) error {
	if err := st.RemoveFund(code); err != nil {
		return err
	}
	if err := st.RemoveAsset(model.AssetKindFund, "", code); err != nil {
		return err
	}
	cfg.RemoveAsset("fund", "", code)
	return cfg.Save(configPath)
}
