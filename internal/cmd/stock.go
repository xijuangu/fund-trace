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
	Use:   "add <code> | add <market> <code>",
	Short: "Add a stock by code (A-share 6-digit or HK 5-digit)",
	Long: `Add a stock by its code.
Market is auto-inferred: 5-digit → hk, 6-digit 6→sh, 0/3→sz.
Specify market explicitly: fund-trace stock add hk 00700`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var market, code string
		if len(args) == 2 {
			market = args[0]
			code = args[1]
		} else {
			code = args[0]
		}
		if market == "" {
			var err error
			market, err = model.InferStockMarket(code)
			if err != nil {
				return err
			}
		}
		if market != "sh" && market != "sz" && market != "hk" {
			return fmt.Errorf("unknown market %q (expected sh, sz, or hk)", market)
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
	Use:     "remove <code> | remove <market> <code>",
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
