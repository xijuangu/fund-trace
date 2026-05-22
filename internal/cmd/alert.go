package cmd

import (
	"fmt"
	"strconv"

	"fund-trace/internal/model"
	"fund-trace/internal/tui"

	"github.com/spf13/cobra"
)

var (
	alertDrop float64
	alertRise float64
)

var alertCmd = &cobra.Command{
	Use:   "alert",
	Short: "Manage price alerts for funds and stocks",
}

var alertSetCmd = &cobra.Command{
	Use:   "set <fund-code> | set stock <code> | set stock <market> <code>",
	Short: "Set a price alert for a fund or stock",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if alertDrop == 0 && alertRise == 0 {
			return fmt.Errorf("must specify --drop or --rise (e.g., --drop 3)")
		}

		isStock := args[0] == "stock"
		var a model.Alert

		if isStock {
			codeArgs := args[1:]
			if len(codeArgs) == 0 || len(codeArgs) > 2 {
				return fmt.Errorf("stock alert: usage: alert set stock <code> | alert set stock <market> <code>")
			}
			var market, code string
			if len(codeArgs) == 2 {
				market = codeArgs[0]
				code = codeArgs[1]
			} else {
				code = codeArgs[0]
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
			a.Kind = model.AssetKindStock
			a.Market = market
			a.Code = code
			a.FundCode = code
		} else {
			code := args[0]
			if len(code) != 6 {
				return fmt.Errorf("invalid code %q: must be 6 digits", code)
			}
			a.Kind = model.AssetKindFund
			a.FundCode = code
			a.Code = code
		}

		a.Enabled = true
		if alertDrop != 0 {
			a.Type = model.AlertDrop
			a.ThresholdPct = -alertDrop
		} else {
			a.Type = model.AlertRise
			a.ThresholdPct = alertRise
		}

		id, err := st.UpsertAlert(a)
		if err != nil {
			return fmt.Errorf("set alert: %w", err)
		}

		if isStock {
			fmt.Printf("Alert #%d set: %s%s ", id, a.Market, a.Code)
		} else {
			fmt.Printf("Alert #%d set: %s ", id, a.FundCode)
		}
		if alertDrop != 0 {
			fmt.Printf("will notify on %.1f%% drop\n", alertDrop)
		} else {
			fmt.Printf("will notify on %.1f%% rise\n", alertRise)
		}
		return nil
	},
}

var alertListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		alerts, err := st.ListAlerts()
		if err != nil {
			return fmt.Errorf("list alerts: %w", err)
		}
		if len(alerts) == 0 {
			fmt.Println("No alerts configured.")
			return nil
		}
		fmt.Printf("\n=== Configured Alerts ===\n\n")
		fmt.Printf("%-4s %-6s %-8s %-8s %-6s %10s %s\n", "ID", "Kind", "Market", "Code", "Type", "Threshold", "Status")
		fmt.Println("──────────────────────────────────────────────────────")
		for _, a := range alerts {
			typeStr := "drop"
			if a.Type == model.AlertRise {
				typeStr = "rise"
			}
			kindStr := a.Kind.String()
			displayCode := a.Code
			if a.Kind == model.AssetKindFund && displayCode == "" {
				displayCode = a.FundCode
			}
			statusStr := "active"
			if !a.Enabled {
				statusStr = tui.ZeroStyle.Render("disabled")
			} else {
				statusStr = tui.PositiveStyle.Render("active")
			}
			fmt.Printf("%-4d %-6s %-8s %-8s %-6s %+9.1f%% %s\n",
				a.ID, kindStr, a.Market, displayCode, typeStr, a.ThresholdPct, statusStr)
		}
		fmt.Println()
		return nil
	},
}

var alertRemoveCmd = &cobra.Command{
	Use:     "remove <alert-id>",
	Aliases: []string{"rm"},
	Short:   "Remove an alert by ID",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid alert ID: %w", err)
		}
		if err := st.DeleteAlert(id); err != nil {
			return fmt.Errorf("remove alert: %w", err)
		}
		fmt.Printf("Removed alert #%d\n", id)
		return nil
	},
}

func init() {
	alertSetCmd.Flags().Float64Var(&alertDrop, "drop", 0, "alert on drop by this percentage (e.g., 3)")
	alertSetCmd.Flags().Float64Var(&alertRise, "rise", 0, "alert on rise by this percentage (e.g., 5)")
	alertCmd.AddCommand(alertSetCmd, alertListCmd, alertRemoveCmd)
}
