package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"fund-trace/internal/model"
)

// RenderFundTable renders a colored table of real-time fund data.
// navHistory is optional: if non-nil, the Trend column shows sparklines
// using the historical NAV values for each fund.
func RenderFundTable(funds []model.RealTimeFund, navHistory map[string][]float64) string {
	if len(funds) == 0 {
		return LoadingStyle.Render("  Loading fund data...")
	}

	var sb strings.Builder

	// Column widths (in terminal display columns, CJK-aware).
	const (
		codeW   = 8
		nameW   = 24
		navW    = 10
		changeW = 12
		trendW  = 10
	)

	// ------ Header ------
	sb.WriteString(HeaderStyle.Render(
		padRight("Code", codeW) + "  " +
			padRight("Name", nameW) + "  " +
			padRight("NAV", navW) + "  " +
			padRight("Change %", changeW) + "  " +
			padRight("Trend", trendW),
	))
	sb.WriteString("\n")

	// Separator.
	sepLen := codeW + nameW + navW + changeW + trendW + 8 // +8 for gaps
	sb.WriteString(strings.Repeat("─", sepLen))
	sb.WriteString("\n")

	// ------ Rows ------
	for _, f := range funds {
		name := truncateByWidth(f.Name, nameW)

		navStr := fmt.Sprintf("%.4f", f.EstimatedNAV)
		if !f.Available {
			navStr = "—"
		}

		changeStr := RenderChange(f.DailyChangePct)

		trendStr := "—"
		if navHistory != nil {
			if history, ok := navHistory[f.Code]; ok {
				spark := Sparkline(history, trendW)
				trendStr = ColorizeChange(f.DailyChangePct, spark)
			}
		}

		sb.WriteString(
			padRight(f.Code, codeW) + "  " +
				padRight(name, nameW) + "  " +
				padRight(navStr, navW) + "  " +
				padRight(changeStr, changeW) + "  " +
				padRight(trendStr, trendW) + "\n",
		)
	}

	return sb.String()
}

// displayWidth returns the number of terminal columns a string occupies.
// Chinese characters and other wide runes count as 2 columns.
func displayWidth(s string) int {
	return lipgloss.Width(s)
}

// padRight returns s padded with trailing spaces to reach 'width' display columns.
func padRight(s string, width int) string {
	dw := displayWidth(s)
	if dw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-dw)
}

// padLeft returns s padded with leading spaces to reach 'width' display columns.
func padLeft(s string, width int) string {
	dw := displayWidth(s)
	if dw >= width {
		return s
	}
	return strings.Repeat(" ", width-dw) + s
}

// truncateByWidth truncates a string so that its display width does not exceed maxWidth.
// If truncation occurs, the last display column is replaced with '…'.
func truncateByWidth(s string, maxWidth int) string {
	if displayWidth(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for i := len(runes) - 1; i >= 1; i-- {
		candidate := string(runes[:i]) + "…"
		if displayWidth(candidate) <= maxWidth {
			return candidate
		}
	}
	return ""
}
