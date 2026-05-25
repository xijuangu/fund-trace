package cmd

import (
	"fmt"
	"os"
	"sort"

	"fund-trace/internal/model"
	"fund-trace/internal/tui"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tracked funds and stocks with current real-time data",
	RunE: func(cmd *cobra.Command, args []string) error {
		codes := cfg.FundCodes()
		funds := fc.FetchAllRealTime(codes)
		dbFunds, _ := st.ListFunds()
		dbNames := make(map[string]string, len(dbFunds))
		for _, f := range dbFunds {
			dbNames[f.Code] = f.Name
		}

		var rtFunds []model.RealTimeFund
		for _, code := range codes {
			if rt, ok := funds[code]; ok && rt != nil {
				r := *rt
				if r.Name == "" {
					r.Name = dbNames[code]
				}
				rtFunds = append(rtFunds, r)
			} else {
				rtFunds = append(rtFunds, model.RealTimeFund{
					Code:      code,
					Name:      dbNames[code],
					Available: false,
				})
			}
		}
		sort.Slice(rtFunds, func(i, j int) bool {
			return rtFunds[i].Code < rtFunds[j].Code
		})

		termW, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || termW == 0 {
			termW = 120
		}
		fmt.Print(tui.RenderFundTable(rtFunds, nil, -1, termW))
		fmt.Println()

		stockEntries := cfg.StockEntries()
		if len(stockEntries) > 0 {
			symbols := make([]string, len(stockEntries))
			for i, e := range stockEntries {
				symbols[i] = e.Market + e.Code
			}
			quotes, err := fc.FetchStockQuotes(symbols)
			if err != nil {
				fmt.Printf("Stocks: fetch error: %v\n", err)
			} else {
				fmt.Println("=== Stocks ===")
				fmt.Printf("%-4s %-8s %-20s %10s %10s %10s %10s\n", "Mkt", "Code", "Name", "Price", "Prev", "Chg%", "Update")
				fmt.Println("──────────────────────────────────────────────────────────────────────────")
				sortedSym := make([]string, 0, len(quotes))
				for s := range quotes {
					sortedSym = append(sortedSym, s)
				}
				sort.Strings(sortedSym)
				for _, sym := range sortedSym {
					q := quotes[sym]
					chg := fmt.Sprintf("%.2f%%", q.ChangePct)
					if q.ChangePct > 0 {
						chg = tui.PositiveStyle.Render(chg)
					} else if q.ChangePct < 0 {
						chg = tui.NegativeStyle.Render(chg)
					}
					fmt.Printf("%-4s %-8s %-20s %10.2f %10.2f %s %10s\n",
						q.Market, q.Code, q.Name, q.Value, q.Previous, chg, q.UpdateTime)
				}
			}
		}
		return nil
	},
}
