package cmd

import (
	"fmt"
	"math"
	"sort"

	"fund-trace/internal/analysis"
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
		if isStockCode(code) {
			return fmt.Errorf("stock history is not yet implemented for %s", code)
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

func init() {
	historyCmd.Flags().IntVar(&historyDays, "days", 30, "number of days of history to fetch")
}

func isStockCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	return code[0] == '0' || code[0] == '3' || code[0] == '6'
}
