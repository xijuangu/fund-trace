package tui

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fund-trace/internal/fetcher"
	"fund-trace/internal/model"
	"fund-trace/internal/notifier"
	"fund-trace/internal/store"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---- Messages ----

// tickMsg is sent by the auto-refresh ticker on every interval.
type tickMsg time.Time

// heartbeatMsg is sent every second to keep the status bar countdown live.
type heartbeatMsg time.Time

// dataFetchedMsg carries the combined results of one fetch cycle.
type dataFetchedMsg struct {
	funds      map[string]*model.RealTimeFund
	navHistory map[string][]float64
	err        error
}

// ---- Config ----

// Config holds TUI-specific settings without importing the config package.
type Config struct {
	RefreshInterval time.Duration
	FundCodes       []string
}

// ---- Model ----

// Model is the main Bubble Tea model for the fund-trace dashboard.
type Model struct {
	store    *store.Store
	fetcher  *fetcher.Client
	notifier *notifier.Notifier
	config   Config

	fundList   []model.Fund
	realtime   map[string]*model.RealTimeFund
	navHistory map[string][]float64 // fund code → historical NAV series for sparklines

	width     int
	height    int
	err       error
	lastFetch time.Time
	loading   bool
	quitting  bool
}

// NewDashboard creates a ready-to-run Bubble Tea Model.
// codes and refreshInterval come from the caller (typically main / CLI).
func NewDashboard(
	st *store.Store,
	fc *fetcher.Client,
	nf *notifier.Notifier,
	codes []string,
	refreshInterval time.Duration,
) *Model {
	funds, err := st.ListFunds()
	if err != nil {
		slog.Error("failed to list funds, falling back to codes", "error", err)
		funds = make([]model.Fund, len(codes))
		for i, code := range codes {
			funds[i] = model.Fund{Code: code}
		}
	}

	return &Model{
		store:      st,
		fetcher:    fc,
		notifier:   nf,
		config: Config{
			RefreshInterval: refreshInterval,
			FundCodes:       codes,
		},
		fundList:   funds,
		realtime:   make(map[string]*model.RealTimeFund),
		navHistory: make(map[string][]float64),
		loading:    true,
	}
}

// ---- tea.Model implementation ----

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchDataCmd(),
		tickCmd(m.config.RefreshInterval),
		heartbeatCmd(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, m.fetchDataCmd()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.loading = true
		return m, tea.Batch(
			tickCmd(m.config.RefreshInterval),
			m.fetchDataCmd(),
		)

	case heartbeatMsg:
		return m, heartbeatCmd()

	case dataFetchedMsg:
		m.lastFetch = time.Now()
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.realtime = msg.funds
			m.navHistory = msg.navHistory
			m.checkAlerts()
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) View() string {
	if m.quitting {
		return TitleStyle.Render("Goodbye!") + "\n"
	}
	if m.width == 0 {
		return "Initializing...\n"
	}

	var sb strings.Builder

	// ---- Header ----
	header := TitleStyle.Render(" Fund Trace ")
	timeStr := StatusStyle.Render(time.Now().Format("2006-01-02 15:04:05"))
	headerLine := lipgloss.JoinHorizontal(lipgloss.Left, header, "    ", timeStr)
	sb.WriteString(headerLine)
	sb.WriteString("\n\n")

	// ---- Error banner ----
	if m.err != nil {
		sb.WriteString(ErrorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		sb.WriteString("\n\n")
	}

	// ---- Table body ----
	if m.loading && len(m.realtime) == 0 {
		sb.WriteString(LoadingStyle.Render("  Fetching fund data..."))
	} else {
		rtFunds := m.resolveFundList()
		sb.WriteString(RenderFundTable(rtFunds, m.navHistory))
	}

	// ---- Status bar ----
	sb.WriteString("\n")
	statusParts := m.buildStatusParts()
	statusBar := StatusStyle.Render(strings.Join(statusParts, " | "))
	sb.WriteString(statusBar)
	sb.WriteString("\n")

	// ---- Keybindings hint ----
	sb.WriteString(StatusStyle.Render("[q]uit  [r]efresh"))

	return sb.String()
}

// ---- Internal helpers ----

// resolveFundList builds an ordered slice of RealTimeFund matching m.fundList,
// falling back to default values for funds that haven't been fetched yet.
func (m *Model) resolveFundList() []model.RealTimeFund {
	var rtFunds []model.RealTimeFund
	for _, f := range m.fundList {
		if rt, ok := m.realtime[f.Code]; ok && rt != nil {
			rtFunds = append(rtFunds, *rt)
		} else {
			rtFunds = append(rtFunds, model.RealTimeFund{
				Code:      f.Code,
				Name:      f.Name,
				Available: false,
			})
		}
	}
	return rtFunds
}

func (m *Model) buildStatusParts() []string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Last update: %s", m.lastFetch.Format("15:04:05")))

	remaining := m.config.RefreshInterval - time.Since(m.lastFetch)
	if remaining > 0 {
		parts = append(parts, fmt.Sprintf("Next refresh: %.0fs", remaining.Seconds()))
	}

	if m.err != nil {
		parts = append(parts, ErrorStyle.Render("⚠ error"))
	}

	return parts
}

func (m *Model) checkAlerts() {
	if m.notifier == nil || len(m.realtime) == 0 {
		return
	}

	var rtFunds []model.RealTimeFund
	for _, rt := range m.realtime {
		if rt != nil {
			rtFunds = append(rtFunds, *rt)
		}
	}

	alerts, err := m.store.ListAlerts()
	if err != nil {
		return
	}

	triggered := m.notifier.CheckAlerts(rtFunds, alerts)
	if len(triggered) > 0 {
		// Build a name map from our fund list.
		nameMap := make(map[string]string, len(m.fundList))
		for _, f := range m.fundList {
			nameMap[f.Code] = f.Name
		}
		m.notifier.NotifyTriggered(triggered, nameMap)
	}
}

// ---- Commands ----

// tickCmd returns a command that fires once after duration d.
// The caller MUST re-issue this command in Update() on every tickMsg
// to create a continuous tick loop.
func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// heartbeatCmd returns a command that fires every second.
// Unlike tickCmd, this only triggers a View re-render (no data fetch)
// so the countdown in the status bar updates in real time.
func heartbeatCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return heartbeatMsg(t)
	})
}

// fetchDataCmd fetches real-time fund data AND historical NAV data for sparklines.
func (m *Model) fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		funds := m.fetcher.FetchAllRealTime(m.config.FundCodes)

		// Fetch nav history for sparklines (last 30 days, oldest→newest).
		navHist := make(map[string][]float64)
		for _, code := range m.config.FundCodes {
			snaps, err := m.store.GetNavHistory(code, 30)
			if err != nil || len(snaps) == 0 {
				continue
			}
			// Nav history comes back descending by date; reverse to oldest→newest.
			values := make([]float64, len(snaps))
			for i, snap := range snaps {
				values[len(snaps)-1-i] = snap.UnitNAV
			}
			navHist[code] = values
		}

		return dataFetchedMsg{
			funds:      funds,
			navHistory: navHist,
		}
	}
}

// Compile-time interface check.
var _ tea.Model = (*Model)(nil)
