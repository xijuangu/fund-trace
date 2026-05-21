package cmd

import (
	"fmt"
	"strings"

	"fund-trace/internal/model"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <fund-code>",
	Short: "Add a fund to the tracking list",
	Long: `Add a fund by its 6-digit code. Automatically discovers the fund name
from East Money's fund database.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code := args[0]
		if len(code) != 6 {
			return fmt.Errorf("invalid fund code %q: must be 6 digits", code)
		}
		// Try to discover the name
		nameMap, err := fc.BuildFundNameMap()
		if err != nil {
			// Non-fatal: just add without name
			fmt.Printf("Warning: could not discover fund name: %v\n", err)
		}
		name := ""
		if n, ok := nameMap[code]; ok {
			name = n
		}
		// Determine fund type from name
		fundType := detectFundType(name)

		if err := st.AddFundWithName(code, name, fundType); err != nil {
			return fmt.Errorf("add fund: %w", err)
		}
		if name != "" {
			fmt.Printf("Added fund %s: %s\n", code, name)
		} else {
			fmt.Printf("Added fund %s (name unknown)\n", code)
		}
		return nil
	},
}

func detectFundType(name string) model.FundType {
	if len(name) == 0 {
		return model.FundUnknown
	}
	// Simple heuristic based on fund name keywords
	if strings.Contains(name, "指数") || strings.Contains(name, "ETF") {
		return model.FundIndex
	}
	if strings.Contains(name, "债券") || strings.Contains(name, "债") || strings.Contains(name, "纯债") {
		return model.FundBond
	}
	if strings.Contains(name, "股票") {
		return model.FundStock
	}
	if strings.Contains(name, "混合") || strings.Contains(name, "灵活") || strings.Contains(name, "平衡") {
		return model.FundMixed
	}
	return model.FundUnknown
}
