package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"fund-trace/internal/config"
	"fund-trace/internal/fetcher"
	"fund-trace/internal/model"
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

	fundCodes []string
	stocks    []struct{ Market, Code string }
)

var rootCmd = &cobra.Command{
	Use:   "fund-trace",
	Short: "A high-performance Chinese fund & stock tracking CLI",
	Long: `fund-trace fetches real-time valuations for Chinese mutual funds
and A-share stocks from 天天基金, 东方财富, and 腾讯财经 APIs.

Default behavior: launches an interactive TUI dashboard with auto-refresh.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return loadDeps()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		refresh := time.Duration(cfg.Settings.RefreshIntervalSec) * time.Second
		dash := tui.NewDashboard(st, fc, nf, fundCodes, stocks, refresh, cfg, configPath)
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

	fundCodes, stocks = cfg.AllAssetCodes()

	st, err = store.Open("fund-trace.db")
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	if err := st.Migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	for _, code := range fundCodes {
		_ = st.AddAssetSimple(model.AssetKindFund, "", code)
	}
	for _, s := range stocks {
		_ = st.AddAssetSimple(model.AssetKindStock, s.Market, s.Code)
	}

	fc = fetcher.New(cfg.Settings.MaxConcurrentRequests)
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
	rootCmd.AddCommand(listCmd, addCmd, removeCmd, historyCmd, alertCmd, exportCmd, monitorCmd, stockCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
