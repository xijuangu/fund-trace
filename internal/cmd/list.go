package cmd

import (
	"fmt"
	"sort"

	"fund-trace/internal/model"
	"fund-trace/internal/tui"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tracked funds with current real-time data",
	RunE: func(cmd *cobra.Command, args []string) error {
		codes := make([]string, len(cfg.Funds))
		for i, f := range cfg.Funds {
			codes[i] = f.Code
		}
		funds := fc.FetchAllRealTime(codes)

		var rtFunds []model.RealTimeFund
		for _, code := range codes {
			if rt, ok := funds[code]; ok && rt != nil {
				rtFunds = append(rtFunds, *rt)
			} else {
				rtFunds = append(rtFunds, model.RealTimeFund{
					Code:      code,
					Available: false,
				})
			}
		}
		sort.Slice(rtFunds, func(i, j int) bool {
			return rtFunds[i].Code < rtFunds[j].Code
		})
		fmt.Print(tui.RenderFundTable(rtFunds, nil))
		return nil
	},
}
