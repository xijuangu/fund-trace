package cmd

import (
	"fmt"

	"fund-trace/internal/model"

	"github.com/spf13/cobra"
)

var stockCmd = &cobra.Command{
	Use:   "stock",
	Short: "Manage tracked stocks",
}

var stockAddCmd = &cobra.Command{
	Use:   "add <code> [market]",
	Short: "Add an A-share stock by 6-digit code",
	Long: `Add a Chinese A-share stock by its 6-digit code.
Market is auto-inferred: codes starting with 6 → sh, 0 or 3 → sz.
Specify market explicitly: fund-trace stock add sh 600519`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var market, code string
		if len(args) == 2 {
			market = args[0]
			code = args[1]
		} else {
			code = args[0]
		}
		if len(code) != 6 {
			return fmt.Errorf("invalid stock code %q: must be 6 digits", code)
		}
		if market == "" {
			var err error
			market, err = model.InferStockMarket(code)
			if err != nil {
				return err
			}
		}
		if market != "sh" && market != "sz" {
			return fmt.Errorf("unknown market %q (expected sh or sz)", market)
		}

		if err := st.AddAssetSimple(model.AssetKindStock, market, code); err != nil {
			return fmt.Errorf("add stock: %w", err)
		}
		cfg.AddStock(market, code)
		if err := cfg.Save(configPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Added stock %s:%s\n", market, code)
		return nil
	},
}

var stockRemoveCmd = &cobra.Command{
	Use:     "remove <code> [market]",
	Aliases: []string{"rm"},
	Short:   "Remove a tracked stock",
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var market, code string
		if len(args) == 2 {
			market = args[0]
			code = args[1]
		} else {
			code = args[0]
		}
		if len(code) != 6 {
			return fmt.Errorf("invalid stock code %q: must be 6 digits", code)
		}
		if market == "" {
			var err error
			market, err = model.InferStockMarket(code)
			if err != nil {
				return err
			}
		}

		if err := st.RemoveAsset(model.AssetKindStock, market, code); err != nil {
			return fmt.Errorf("remove stock: %w", err)
		}
		cfg.RemoveAsset("stock", market, code)
		if err := cfg.Save(configPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Removed stock %s:%s\n", market, code)
		return nil
	},
}

func init() {
	stockCmd.AddCommand(stockAddCmd, stockRemoveCmd)
}
