package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"time"

	"fund-trace/internal/model"

	"github.com/spf13/cobra"
)

var exportFormat string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export fund data (CSV or HTML)",
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
				rtFunds = append(rtFunds, model.RealTimeFund{Code: code, Available: false})
			}
		}
		sort.Slice(rtFunds, func(i, j int) bool {
			return rtFunds[i].Code < rtFunds[j].Code
		})

		switch exportFormat {
		case "csv":
			return exportCSV(rtFunds)
		case "html":
			return exportHTML(rtFunds)
		default:
			return fmt.Errorf("unsupported format: %s (use csv or html)", exportFormat)
		}
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "csv", "export format: csv or html")
}

func exportCSV(funds []model.RealTimeFund) error {
	now := time.Now()
	filename := fmt.Sprintf("fund-data-%04d-%02d-%02d.csv", now.Year(), now.Month(), now.Day())
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create csv: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write([]string{"Code", "Name", "NAV", "Previous NAV", "Change %", "Update Time"})
	for _, rtf := range funds {
		nav := ""
		prev := ""
		change := ""
		if rtf.Available {
			nav = fmt.Sprintf("%.4f", rtf.EstimatedNAV)
			prev = fmt.Sprintf("%.4f", rtf.PreviousNAV)
			change = fmt.Sprintf("%.2f%%", rtf.DailyChangePct)
		}
		w.Write([]string{rtf.Code, rtf.Name, nav, prev, change, rtf.UpdateTime})
	}
	w.Flush()
	fmt.Printf("Exported to %s\n", filename)
	return nil
}

func exportHTML(funds []model.RealTimeFund) error {
	now := time.Now()
	filename := fmt.Sprintf("fund-data-%04d-%02d-%02d.html", now.Year(), now.Month(), now.Day())
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create html: %w", err)
	}
	defer f.Close()

	f.WriteString("<html><head><meta charset='utf-8'><title>Fund Data</title>")
	f.WriteString("<style>body{font-family:sans-serif} table{border-collapse:collapse} td,th{border:1px solid #ccc;padding:8px} .up{color:green} .down{color:red}</style>")
	f.WriteString("</head><body><h1>Fund Data</h1><table><tr><th>Code</th><th>Name</th><th>NAV</th><th>Change</th><th>Update</th></tr>")
	for _, rtf := range funds {
		cls := ""
		if rtf.DailyChangePct > 0 {
			cls = "up"
		} else if rtf.DailyChangePct < 0 {
			cls = "down"
		}
		nav := ""
		change := ""
		if rtf.Available {
			nav = fmt.Sprintf("%.4f", rtf.EstimatedNAV)
			change = fmt.Sprintf("<span class='%s'>%.2f%%</span>", cls, rtf.DailyChangePct)
		} else {
			nav = "—"
			change = "—"
		}
		f.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			rtf.Code, rtf.Name, nav, change, rtf.UpdateTime))
	}
	f.WriteString("</table></body></html>")
	fmt.Printf("Exported to %s\n", filename)
	return nil
}
