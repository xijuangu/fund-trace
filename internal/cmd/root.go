package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"fund-trace/internal/config"
	"fund-trace/internal/fetcher"
	"fund-trace/internal/notifier"
	"fund-trace/internal/store"
	"fund-trace/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	configPath string
	cfg        *config.Config
	st         *store.Store
	fc         *fetcher.Client
	nf         *notifier.Notifier
)

var rootCmd = &cobra.Command{
	Use:   "fund-trace",
	Short: "A high-performance Chinese mutual fund tracking CLI",
	Long: `fund-trace fetches real-time valuations and historical NAV data
for Chinese mutual funds from 天天基金 and 东方财富 APIs.

Default behavior: launches an interactive TUI dashboard with auto-refresh.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return loadDeps()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: launch TUI dashboard
		codes := make([]string, len(cfg.Funds))
		for i, f := range cfg.Funds {
			codes[i] = f.Code
		}
		refresh := time.Duration(cfg.Settings.RefreshIntervalSec) * time.Second
		dash := tui.NewDashboard(st, fc, nf, codes, refresh, cfg, configPath)
		p := tea.NewProgram(dash, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func loadDeps() error {
	var err error
	cfg, err = config.LoadOrCreate(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	st, err = store.Open("fund-trace.db")
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	if err := st.Migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	// Seed funds from config
	codes := make([]string, len(cfg.Funds))
	for i, f := range cfg.Funds {
		codes[i] = f.Code
	}
	if err := st.SeedFromConfig(codes); err != nil {
		slog.Warn("seed funds", "error", err)
	}

	fc = fetcher.New(cfg.Settings.MaxConcurrentRequests)
	// Fill in missing fund names from the API (one-time startup cost).
	fillMissingNames(st, fc)
	nf = notifier.New(time.Duration(cfg.Settings.AlertCooldownMin) * time.Minute)
	return nil
}

func fillMissingNames(st *store.Store, fc *fetcher.Client) {
	funds, err := st.ListFunds()
	if err != nil {
		return
	}
	missing := false
	for _, f := range funds {
		if f.Name == "" {
			missing = true
			break
		}
	}
	if !missing {
		return
	}
	nameMap, err := fc.BuildFundNameMap()
	if err != nil {
		slog.Warn("fill names: failed to fetch fund list", "error", err)
		return
	}
	for _, f := range funds {
		if f.Name == "" {
			if name, ok := nameMap[f.Code]; ok {
				if err := st.UpdateFundName(f.Code, name); err != nil {
					slog.Warn("fill names: update failed", "code", f.Code, "error", err)
				}
			}
		}
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yaml", "path to config file")
	rootCmd.AddCommand(listCmd, addCmd, removeCmd, historyCmd, alertCmd, exportCmd, monitorCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
