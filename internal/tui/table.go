package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"fund-trace/internal/model"
)

// RenderAssetTable renders a colored table of mixed fund and stock data.
// navHistory is optional: if non-nil, the Trend column shows sparklines
// using the historical NAV values for each fund. Stocks always show "—".
func RenderAssetTable(rows []AssetRow, navHistory map[string][]float64, cursor int) string {
	if len(rows) == 0 {
		return LoadingStyle.Render("  Loading asset data...")
	}

	var sb strings.Builder

	// Column widths (in terminal display columns, CJK-aware).
	const (
		typeW   = 6
		mktW    = 4
		codeW   = 8
		nameW   = 18
		valueW  = 10
		changeW = 10
		trendW  = 10
	)

	// ------ Header ------
	sb.WriteString(HeaderStyle.Render(
		padRight("Type", typeW) + "  " +
			padRight("Mkt.", mktW) + "  " +
			padRight("Code", codeW) + "  " +
			padRight("Name", nameW) + "  " +
			padRight("Price/NAV", valueW) + "  " +
			padRight("Change%", changeW) + "  " +
			padRight("Trend", trendW),
	))
	sb.WriteString("\n")

	// Separator.
	sepLen := typeW + mktW + codeW + nameW + valueW + changeW + trendW + 12
	sb.WriteString(strings.Repeat("─", sepLen))
	sb.WriteString("\n")

	// ------ Rows ------
	for i, r := range rows {
		typeStr := "Fund"
		mktStr := "—"
		if r.Kind == model.AssetKindStock {
			typeStr = "Stock"
			mktStr = r.Market
		}

		name := truncateByWidth(r.Name, nameW)

		var valueStr string
		isAvailable := r.Available
		if !isAvailable {
			valueStr = "—"
		} else if r.Kind == model.AssetKindStock {
			valueStr = fmt.Sprintf("%.2f", r.Value)
		} else {
			valueStr = fmt.Sprintf("%.4f", r.Value)
		}

		changeStr := "—"
		if isAvailable {
			changeStr = RenderChange(r.ChangePct)
		}

		trendStr := "—"
		if r.Kind == model.AssetKindFund && navHistory != nil {
			if history, ok := navHistory[r.Code]; ok {
				blocks := Sparkline(history, trendW)
				var sb2 strings.Builder
				for _, b := range blocks {
					if b.Char == '▄' {
						sb2.WriteString(ZeroStyle.Render(string(b.Char)))
					} else {
						sb2.WriteString(ColorizeChange(b.Value, string(b.Char)))
					}
				}
				trendStr = sb2.String()
			}
		}

		row := padRight(typeStr, typeW) + "  " +
			padRight(mktStr, mktW) + "  " +
			padRight(r.Code, codeW) + "  " +
			padRight(name, nameW) + "  " +
			padRight(valueStr, valueW) + "  " +
			padRight(changeStr, changeW) + "  " +
			padRight(trendStr, trendW)

		if i == cursor {
			row = "\033[7m" + row + "\033[0m"
		}
		sb.WriteString(row + "\n")
	}

	return sb.String()
}

// RenderFundTable renders a colored table of real-time fund data.
// navHistory is optional: if non-nil, the Trend column shows sparklines
// using the historical NAV values for each fund.
func RenderFundTable(funds []model.RealTimeFund, navHistory map[string][]float64, cursor int) string {
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
	for i, f := range funds {
		name := truncateByWidth(f.Name, nameW)

		navStr := fmt.Sprintf("%.4f", f.EstimatedNAV)
		if !f.Available {
			navStr = "—"
		}

		changeStr := RenderChange(f.DailyChangePct)

		trendStr := "—"
		if navHistory != nil {
			if history, ok := navHistory[f.Code]; ok {
				blocks := Sparkline(history, trendW)
				var sb2 strings.Builder
				for _, b := range blocks {
					if b.Char == '▄' {
						sb2.WriteString(ZeroStyle.Render(string(b.Char)))
					} else {
						sb2.WriteString(ColorizeChange(b.Value, string(b.Char)))
					}
				}
				trendStr = sb2.String()
			}
		}

		row := padRight(f.Code, codeW) + "  " +
			padRight(name, nameW) + "  " +
			padRight(navStr, navW) + "  " +
			padRight(changeStr, changeW) + "  " +
			padRight(trendStr, trendW)

		if i == cursor {
			row = "\033[7m" + row + "\033[0m"
		}
		sb.WriteString(row + "\n")
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
