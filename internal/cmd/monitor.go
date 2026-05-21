package cmd

import (
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:     "monitor",
	Aliases: []string{"mon"},
	Short:   "Launch the interactive TUI dashboard (same as default)",
	RunE:    rootCmd.RunE,
}
