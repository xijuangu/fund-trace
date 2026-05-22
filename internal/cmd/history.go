package cmd

import (
	"fmt"
	"math"
	"sort"

	"fund-trace/internal/analysis"
	"fund-trace/internal/fetcher"
	"fund-trace/internal/model"
	"fund-trace/internal/store"
	"fund-trace/internal/tui"

	"github.com/spf13/cobra"
)

var historyDays int

var historyCmd = &cobra.Command{
	Use:   "history <fund-code>",
	Short: "Show historical NAV data with trend analysis",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code := args[0]
		if len(code) != 6 {
			return fmt.Errorf("invalid code %q: must be 6 digits", code)
		}
		if isStock, market, _ := resolveStockHistoryRequest(cfg, code); isStock {
			return showStockHistory(st, fc, market, code, historyDays)
		}

		snapshots, err := st.GetNavHistory(code, historyDays)
		if err != nil || len(snapshots) == 0 {
			snapshots, err = fc.FetchHistory(code, historyDays)
			if err != nil {
				return fmt.Errorf("fetch history for %s: %w", code, err)
			}
			_ = st.SaveNavSnapshots(snapshots)
		}
		if len(snapshots) == 0 {
			return fmt.Errorf("no history data for fund %s", code)
		}

		sort.Slice(snapshots, func(i, j int) bool {
			return snapshots[i].Date < snapshots[j].Date
		})

		tr := analysis.TrendSummary(snapshots)

		recent := snapshots
		if len(recent) > 10 {
			recent = recent[len(recent)-10:]
		}
		sort.Slice(recent, func(i, j int) bool {
			return recent[i].Date > recent[j].Date
		})

		fmt.Printf("\n=== History: %s (%d days) ===\n\n", code, historyDays)
		fmt.Printf("%-12s %10s %10s\n", "Date", "NAV", "Change%")
		fmt.Println("─────────────────────────────────────")
		for _, s := range recent {
			changeStr := fmt.Sprintf("%+.2f%%", s.DailyGrowthPct)
			if s.DailyGrowthPct > 0 {
				changeStr = tui.PositiveStyle.Render(changeStr)
			} else if s.DailyGrowthPct < 0 {
				changeStr = tui.NegativeStyle.Render(changeStr)
			}
			fmt.Printf("%-12s %10.4f %s\n", s.Date, s.UnitNAV, changeStr)
		}

		fmt.Println("\n=== Trend Analysis ===")
		fmt.Printf("  Direction:   %s\n", tr.Direction)
		fmt.Printf("  5-day change: %+.2f%%\n", tr.Change5D)
		if sma5 := analysis.Latest(tr.SMA5); !math.IsNaN(sma5) {
			fmt.Printf("  SMA(5):      %.4f\n", sma5)
		}
		if sma20 := analysis.Latest(tr.SMA20); !math.IsNaN(sma20) {
			fmt.Printf("  SMA(20):     %.4f\n", sma20)
		}
		if rsi14 := analysis.Latest(tr.RSI14); !math.IsNaN(rsi14) {
			rsiLabel := "neutral"
			if rsi14 > 70 {
				rsiLabel = "overbought"
			} else if rsi14 < 30 {
				rsiLabel = "oversold"
			}
			fmt.Printf("  RSI(14):     %.2f (%s)\n", rsi14, rsiLabel)
		}
		fmt.Println()
		return nil
	},
}

var historyStockCmd = &cobra.Command{
	Use:   "stock <code> | stock <market> <code>",
	Short: "Show historical price data for a stock with trend analysis",
	Args:  cobra.RangeArgs(1, 2),
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
		return showStockHistory(st, fc, market, code, historyDays)
	},
}

func showStockHistory(st *store.Store, fc *fetcher.Client, market, code string, days int) error {
	snapshots, err := st.GetPriceHistory(model.AssetKindStock, market, code, days)
	if err != nil || len(snapshots) == 0 {
		snapshots, err = fc.FetchStockHistory(market, code, days)
		if err != nil {
			return fmt.Errorf("fetch stock history for %s:%s: %w", market, code, err)
		}
		if len(snapshots) == 0 {
			return fmt.Errorf("no history data for stock %s:%s", market, code)
		}
		_ = st.SavePriceSnapshots(snapshots)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Date < snapshots[j].Date
	})

	closePrices := make([]float64, len(snapshots))
	for i, s := range snapshots {
		closePrices[i] = s.Close
	}
	tr := analysis.TrendSummaryFromValues(closePrices)

	recent := snapshots
	if len(recent) > 10 {
		recent = recent[len(recent)-10:]
	}
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].Date > recent[j].Date
	})

	fmt.Printf("\n=== Stock History: %s%s (%d days) ===\n\n", market, code, days)
	fmt.Printf("%-12s %10s %10s\n", "Date", "Close", "Change%")
	fmt.Println("─────────────────────────────────────")
	for _, s := range recent {
		changeStr := fmt.Sprintf("%+.2f%%", s.ChangePct)
		if s.ChangePct > 0 {
			changeStr = tui.PositiveStyle.Render(changeStr)
		} else if s.ChangePct < 0 {
			changeStr = tui.NegativeStyle.Render(changeStr)
		}
		fmt.Printf("%-12s %10.2f %s\n", s.Date, s.Close, changeStr)
	}

	fmt.Println("\n=== Trend Analysis ===")
	fmt.Printf("  Direction:   %s\n", tr.Direction)
	fmt.Printf("  5-day change: %+.2f%%\n", tr.Change5D)
	if sma5 := analysis.Latest(tr.SMA5); !math.IsNaN(sma5) {
		fmt.Printf("  SMA(5):      %.2f\n", sma5)
	}
	if sma20 := analysis.Latest(tr.SMA20); !math.IsNaN(sma20) {
		fmt.Printf("  SMA(20):     %.2f\n", sma20)
	}
	if rsi14 := analysis.Latest(tr.RSI14); !math.IsNaN(rsi14) {
		rsiLabel := "neutral"
		if rsi14 > 70 {
			rsiLabel = "overbought"
		} else if rsi14 < 30 {
			rsiLabel = "oversold"
		}
		fmt.Printf("  RSI(14):     %.2f (%s)\n", rsi14, rsiLabel)
	}
	fmt.Println()
	return nil
}

func init() {
	historyCmd.PersistentFlags().IntVar(&historyDays, "days", 30, "number of days of history to fetch")
	historyCmd.AddCommand(historyStockCmd)
}
