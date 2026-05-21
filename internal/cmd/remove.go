package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <fund-code>",
	Aliases: []string{"rm"},
	Short:   "Remove a fund from the tracking list",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code := args[0]
		if len(code) != 6 {
			return fmt.Errorf("invalid fund code %q: must be 6 digits", code)
		}
		if err := st.RemoveFund(code); err != nil {
			return fmt.Errorf("remove fund: %w", err)
		}
		fmt.Printf("Removed fund %s\n", code)
		return nil
	},
}
