package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

var backfillDays int
var backfillSleepMs int

var backfillCmd = &cobra.Command{
	Use:   "backfill",
	Short: "Backfill historical NAV data for all tracked funds",
	RunE: func(cmd *cobra.Command, args []string) error {
		if backfillDays <= 0 {
			return fmt.Errorf("--days must be positive")
		}

		funds, err := st.ListFunds()
		if err != nil {
			return fmt.Errorf("list funds: %w", err)
		}
		if len(funds) == 0 {
			return fmt.Errorf("no funds to backfill")
		}

		sort.Slice(funds, func(i, j int) bool {
			return funds[i].Code < funds[j].Code
		})

		var failed []string
		totalRows := 0

		for i, fund := range funds {
			fmt.Printf("[%d/%d] Backfilling %s %s ... ", i+1, len(funds), fund.Code, fund.Name)

			snapshots, err := fc.FetchHistory(fund.Code, backfillDays)
			if err != nil {
				fmt.Printf("failed: %v\n", err)
				failed = append(failed, fund.Code)
				continue
			}

			if err := st.SaveNavSnapshots(snapshots); err != nil {
				fmt.Printf("save failed: %v\n", err)
				failed = append(failed, fund.Code)
				continue
			}

			totalRows += len(snapshots)

			startDate := ""
			endDate := ""
			if len(snapshots) > 0 {
				startDate = snapshots[len(snapshots)-1].Date
				endDate = snapshots[0].Date
			}

			fmt.Printf("ok: %d rows, %s ~ %s\n", len(snapshots), startDate, endDate)

			if backfillSleepMs > 0 {
				time.Sleep(time.Duration(backfillSleepMs) * time.Millisecond)
			}
		}

		fmt.Println()
		fmt.Printf("Backfill complete. fetched rows: %d\n", totalRows)

		if len(failed) > 0 {
			fmt.Printf("Failed funds: %v\n", failed)
			return fmt.Errorf("backfill finished with %d failed funds", len(failed))
		}

		return nil
	},
}

func init() {
	backfillCmd.Flags().IntVar(&backfillDays, "days", 5000, "number of historical NAV rows to fetch per fund")
	backfillCmd.Flags().IntVar(&backfillSleepMs, "sleep-ms", 300, "sleep milliseconds between funds")
}
