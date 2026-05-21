package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	green   = lipgloss.Color("#01AB4F")
	red     = lipgloss.Color("#FF4D52")
	dim     = lipgloss.Color("#626262")
	cyan    = lipgloss.Color("#00BCD4")
	magenta = lipgloss.Color("#E040FB")
)

// Shared styles used across TUI and CLI rendering.
var (
	// HeaderStyle is used for column headers in the fund table.
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyan).
			Padding(0, 1)

	// StatusStyle is used for status bar and timestamp text.
	StatusStyle = lipgloss.NewStyle().
			Foreground(dim).
			Padding(0, 1)

	// TitleStyle is used for the dashboard title header.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(magenta).
			Padding(0, 1)

	// TableStyle wraps the full table with a rounded border.
	TableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(dim)

	// PositiveStyle colors positive change values green.
	PositiveStyle = lipgloss.NewStyle().Foreground(green)

	// NegativeStyle colors negative change values red.
	NegativeStyle = lipgloss.NewStyle().Foreground(red)

	// ZeroStyle colors zero/neutral change values dim.
	ZeroStyle = lipgloss.NewStyle().Foreground(dim)

	// LoadingStyle renders "loading" placeholder text.
	LoadingStyle = lipgloss.NewStyle().Foreground(dim).Italic(true)

	// ErrorStyle renders error messages.
	ErrorStyle = lipgloss.NewStyle().Foreground(red)
)

// RenderChange formats a daily change percentage with sign and color.
// Positive values get "+" prefix and green; negative get red; zero is dim.
func RenderChange(pct float64) string {
	switch {
	case pct > 0:
		return PositiveStyle.Render(fmt.Sprintf("+%.2f%%", pct))
	case pct < 0:
		return NegativeStyle.Render(fmt.Sprintf("%.2f%%", pct))
	default:
		return ZeroStyle.Render("0.00%")
	}
}

// ColorizeChange applies color to a pre-formatted value string based on its sign.
func ColorizeChange(pct float64, value string) string {
	switch {
	case pct > 0:
		return PositiveStyle.Render(value)
	case pct < 0:
		return NegativeStyle.Render(value)
	default:
		return ZeroStyle.Render(value)
	}
}
