package tui

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fund-trace/internal/analysis"
	"fund-trace/internal/config"
	"fund-trace/internal/fetcher"
	"fund-trace/internal/model"
	"fund-trace/internal/notifier"
	"fund-trace/internal/store"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---- Mode ----

type mode int

const (
	modeNormal        mode = iota
	modeAddFund
	modeConfirmDelete
	modeAlertSet
	modeSettings
	modeDetail
	modeHelp
)

// ---- Messages ----

// tickMsg is sent by the auto-refresh ticker on every interval.
type tickMsg time.Time

// heartbeatMsg is sent every second to keep the status bar countdown live.
type heartbeatMsg time.Time

// dataFetchedMsg carries the combined results of one fetch cycle.
type dataFetchedMsg struct {
	funds       map[string]*model.RealTimeFund
	stockQuotes map[string]*model.Quote
	navHistory  map[string][]float64
	err         error
}

// assetAddedMsg carries the result of adding a new fund or stock.
type assetAddedMsg struct {
	kind   model.AssetKind
	market string
	code   string
	name   string
	err    error
}

// detailFetchedMsg carries historical data and trend analysis for a fund.
type detailFetchedMsg struct {
	snapshots []model.NavSnapshot
	trend     analysis.TrendResult
	err       error
}

// ---- Config ----

// Config holds TUI-specific settings without importing the config package.
type Config struct {
	RefreshInterval time.Duration
	FundCodes       []string
	StockSymbols    []string
}

// StockEntry holds a stock market+code pair for display.
type StockEntry struct {
	Market string
	Code   string
}

// AssetRow is a unified row for displaying either a fund or stock in the table.
type AssetRow struct {
	Kind       model.AssetKind
	Market     string
	Code       string
	Name       string
	Available  bool
	Value      float64
	Previous   float64
	ChangePct  float64
	UpdateTime string
}

// ---- Model ----

// Model is the main Bubble Tea model for the fund-trace dashboard.
type Model struct {
	store    *store.Store
	fetcher  *fetcher.Client
	notifier *notifier.Notifier
	config   Config

	appConfig  *config.Config
	configPath string

	assetList   []model.Asset
	stockList   []model.Asset
	stockSym    []string
	realtime    map[string]*model.RealTimeFund
	stockQuotes map[string]*model.Quote
	navHistory  map[string][]float64

	mode          mode
	cursor        int
	textInput     textinput.Model
	confirmTarget *model.Asset
	alertTarget   *model.Asset
	alertIsRise   bool
	settingsIdx        int
	settingsEditing    bool
	settingsEditInput  textinput.Model
	detailAsset        *model.Asset
	detailSnapshots    []model.NavSnapshot
	detailTrend        analysis.TrendResult
	detailLoading      bool

	width     int
	height    int
	err       error
	lastFetch time.Time
	loading   bool
	quitting  bool
}

func newTextInput() textinput.Model {
	ti := textinput.New()
	ti.Width = 20
	ti.CharLimit = 6
	ti.Placeholder = "000000"
	return ti
}

// NewDashboard creates a ready-to-run Bubble Tea Model.
func NewDashboard(
	st *store.Store,
	fc *fetcher.Client,
	nf *notifier.Notifier,
	codes []string,
	stocks []struct{ Market, Code string },
	refreshInterval time.Duration,
	appCfg *config.Config,
	cfgPath string,
) *Model {
	assets, err := st.ListAssets()
	if err != nil {
		slog.Error("failed to list assets, falling back to codes", "error", err)
		assets = make([]model.Asset, 0, len(codes)+len(stocks))
		for _, code := range codes {
			assets = append(assets, model.Asset{Kind: model.AssetKindFund, Code: code})
		}
		for _, s := range stocks {
			assets = append(assets, model.Asset{Kind: model.AssetKindStock, Market: s.Market, Code: s.Code})
		}
	}

	stockSyms := make([]string, 0, len(stocks))
	for _, s := range stocks {
		stockSyms = append(stockSyms, s.Market+s.Code)
	}

	return &Model{
		store:      st,
		fetcher:    fc,
		notifier:   nf,
		config: Config{
			RefreshInterval: refreshInterval,
			FundCodes:       codes,
			StockSymbols:    stockSyms,
		},
		appConfig:  appCfg,
		configPath: cfgPath,
		assetList:  assets,
		stockSym:   stockSyms,
		realtime:   make(map[string]*model.RealTimeFund),
		stockQuotes: make(map[string]*model.Quote),
		navHistory: make(map[string][]float64),
		loading:    true,
		mode:       modeNormal,
		cursor:     0,
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.mode == modeNormal {
			m.loading = true
			return m, tea.Batch(tickCmd(m.config.RefreshInterval), m.fetchDataCmd())
		}
		return m, nil

	case heartbeatMsg:
		return m, heartbeatCmd()

	case dataFetchedMsg:
		m.lastFetch = time.Now()
		m.loading = false
		if m.mode == modeNormal {
			if msg.err != nil {
				m.err = msg.err
			} else {
				m.err = nil
				m.realtime = msg.funds
				m.stockQuotes = msg.stockQuotes
				if m.stockQuotes == nil {
					m.stockQuotes = make(map[string]*model.Quote)
				}
				m.navHistory = msg.navHistory
				m.checkAlerts()
			}
		}
		return m, nil

	case assetAddedMsg:
		return m.handleAssetAdded(msg)

	case detailFetchedMsg:
		return m.handleDetailFetched(msg)

	case tea.KeyMsg:
		switch m.mode {
		case modeNormal:
			return m.updateNormal(msg)
		case modeConfirmDelete:
			return m.updateConfirmDelete(msg)
		case modeAddFund:
			return m.updateAddFund(msg)
		case modeAlertSet:
			return m.updateAlertSet(msg)
		case modeSettings:
			return m.updateSettings(msg)
		case modeDetail:
			return m.updateDetail(msg)
		case modeHelp:
			return m.updateHelp(msg)
		}
	}

	return m, nil
}

func (m *Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if len(m.assetList) > 0 {
			m.cursor = min(m.cursor+1, len(m.assetList)-1)
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.quitting = true
		return m, tea.Quit
	case "r":
		m.loading = true
		return m, m.fetchDataCmd()
	case "a":
		return m.enterAddFund()
	case "d":
		return m.enterConfirmDelete()
	case "A", "shift+a":
		return m.enterAlertSet()
	case "s":
		m.mode = modeSettings
		m.settingsIdx = 0
		m.settingsEditing = false
	case "enter":
		return m.enterDetail()
	case "h":
		m.mode = modeHelp
	}
	return m, nil
}

func (m *Model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		target := m.confirmTarget
		kind := target.Kind
		market := target.Market
		code := target.Code

		if kind == model.AssetKindFund {
			m.store.RemoveFund(code)
			m.store.RemoveAsset(model.AssetKindFund, "", code)
			m.config.FundCodes = removeFromSlice(m.config.FundCodes, code)
			m.appConfig.RemoveAsset("fund", "", code)
		} else {
			m.store.RemoveAsset(kind, market, code)
			sym := market + code
			m.config.StockSymbols = removeFromSlice(m.config.StockSymbols, sym)
			m.appConfig.RemoveAsset("stock", market, code)
		}
		m.appConfig.Save(m.configPath)
		for i, a := range m.assetList {
			if a.Code == code && a.Kind == kind && a.Market == market {
				m.assetList = append(m.assetList[:i], m.assetList[i+1:]...)
				break
			}
		}
		m.mode = modeNormal
		if m.cursor >= len(m.assetList) && len(m.assetList) > 0 {
			m.cursor = len(m.assetList) - 1
		}
		return m, m.fetchDataCmd()
	case "n", "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m *Model) updateAlertSet(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "t":
		m.alertIsRise = !m.alertIsRise
		return m, nil
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "enter":
		val := parseFloatOrZero(m.textInput.Value())
		if val == 0 {
			return m, nil
		}
		at := model.AlertDrop
		threshold := -val
		if m.alertIsRise {
			at = model.AlertRise
			threshold = val
		}
		m.store.UpsertAlert(model.Alert{
			FundCode:     m.alertTarget.Code,
			Type:         at,
			ThresholdPct: threshold,
			Enabled:      true,
		})
		m.mode = modeNormal
		return m, nil
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

func (m *Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if !m.settingsEditing {
			m.settingsIdx = min(m.settingsIdx+1, 3)
		}
	case "k", "up":
		if !m.settingsEditing {
			m.settingsIdx = max(m.settingsIdx-1, 0)
		}
	case "enter":
		if !m.settingsEditing {
			m.settingsEditing = true
			ti := newTextInput()
			ti.Width = 10
			ti.CharLimit = 6
			ti.SetValue(m.settingsFieldValue(m.settingsIdx))
			ti.Focus()
			m.settingsEditInput = ti
		}
	case "esc":
		if m.settingsEditing {
			m.settingsEditing = false
		} else {
			m.appConfig.Save(m.configPath)
			m.mode = modeNormal
		}
	default:
		if m.settingsEditing {
			switch msg.String() {
			case "enter":
				val := parseFloatOrZero(m.settingsEditInput.Value())
				m.applySettingsValue(m.settingsIdx, int(val))
				m.settingsEditing = false
			default:
				var cmd tea.Cmd
				m.settingsEditInput, cmd = m.settingsEditInput.Update(msg)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m *Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m *Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = modeNormal
	return m, nil
}

func (m *Model) handleAssetAdded(msg assetAddedMsg) (tea.Model, tea.Cmd) {
	m.mode = modeNormal
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}

	if msg.kind == model.AssetKindFund {
		m.config.FundCodes = append(m.config.FundCodes, msg.code)
		m.appConfig.AddFund(msg.code)
	} else {
		sym := msg.market + msg.code
		m.config.StockSymbols = append(m.config.StockSymbols, sym)
		m.appConfig.AddStock(msg.market, msg.code)
	}
	m.appConfig.Save(m.configPath)

	for _, a := range m.assetList {
		if a.Kind == msg.kind && a.Market == msg.market && a.Code == msg.code {
			m.err = fmt.Errorf("asset already exists: %s/%s", msg.market, msg.code)
			return m, nil
		}
	}
	m.assetList = append(m.assetList, model.Asset{
		Kind:   msg.kind,
		Market: msg.market,
		Code:   msg.code,
		Name:   msg.name,
	})
	m.cursor = len(m.assetList) - 1
	return m, m.fetchDataCmd()
}

func (m *Model) handleDetailFetched(msg detailFetchedMsg) (tea.Model, tea.Cmd) {
	m.detailLoading = false
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}
	m.detailSnapshots = msg.snapshots
	m.detailTrend = msg.trend
	return m, nil
}

func (m *Model) View() string {
	if m.quitting {
		return TitleStyle.Render("Goodbye!") + "\n"
	}
	if m.width == 0 {
		return "Initializing...\n"
	}
	if m.mode == modeDetail {
		return m.detailView()
	}

	base := m.normalView()
	switch m.mode {
	case modeAddFund:
		return m.overlayModal(base, m.addFundView())
	case modeConfirmDelete:
		return m.overlayModal(base, m.confirmDeleteView())
	case modeAlertSet:
		return m.overlayModal(base, m.alertSetView())
	case modeSettings:
		return m.overlayModal(base, m.settingsView())
	case modeHelp:
		return m.overlayModal(base, m.helpView())
	default:
		return base
	}
}

func (m *Model) normalView() string {
	var sb strings.Builder

	header := TitleStyle.Render(" Fund Trace ")
	timeStr := StatusStyle.Render(time.Now().Format("2006-01-02 15:04:05"))
	headerLine := lipgloss.JoinHorizontal(lipgloss.Left, header, "    ", timeStr)
	sb.WriteString(headerLine)
	sb.WriteString("\n\n")

	if m.err != nil {
		sb.WriteString(ErrorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		sb.WriteString("\n\n")
	}

	if m.loading && len(m.realtime) == 0 {
		sb.WriteString(LoadingStyle.Render("  Fetching asset data..."))
	} else {
		rf := m.resolveAssetList()
		sb.WriteString(RenderAssetTable(rf, m.navHistory, m.cursor))
	}

	sb.WriteString("\n")
	statusParts := m.buildStatusParts()
	statusBar := StatusStyle.Render(strings.Join(statusParts, " | "))
	sb.WriteString(statusBar)
	sb.WriteString("\n")

	sb.WriteString(StatusStyle.Render(m.keyHints()))
	return sb.String()
}

func (m *Model) overlayModal(base, modal string) string {
	return base + "\n" + modal
}

func (m *Model) keyHints() string {
	switch m.mode {
	case modeNormal:
		return "[q]uit  [r]efresh  [a]dd  [d]el  [A]lert  [s]ettings  [h]elp  [j/k] nav  [enter] detail"
	case modeAddFund:
		return "[Enter] confirm  [Esc] cancel"
	case modeConfirmDelete:
		return "[y]es delete  [n]o keep"
	case modeAlertSet:
		return "[Enter] confirm  [t]oggle rise/drop  [Esc] back"
	case modeSettings:
		return "[j/k] navigate  [Enter] edit  [Esc] save & back"
	default:
		return "[Esc] back"
	}
}

func (m *Model) addFundView() string {
	var sb strings.Builder
	sb.WriteString(DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			TitleStyle.Render("Add Asset"),
			"",
			m.textInput.View(),
			"",
			StatusStyle.Render("Fund:  6-digit code, e.g. 011513"),
			StatusStyle.Render("Stock: sh600519  /  sz000001  /  stock:sh:600519"),
			StatusStyle.Render("[Enter] confirm  [Esc] cancel"),
		),
	))
	return sb.String()
}

func (m *Model) confirmDeleteView() string {
	name := ""
	typeStr := "Asset"
	if m.confirmTarget != nil {
		n := m.confirmTarget.Name
		c := m.confirmTarget.Code
		if n == "" {
			n = c
		} else {
			n = fmt.Sprintf("%s (%s)", n, c)
		}
		if m.confirmTarget.Kind == model.AssetKindFund {
			typeStr = "Fund"
			name = n
		} else {
			typeStr = "Stock"
			name = fmt.Sprintf("%s:%s — %s", m.confirmTarget.Market, m.confirmTarget.Code, n)
		}
	}
	return DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			TitleStyle.Render("Delete "+typeStr),
			"",
			fmt.Sprintf("  Remove %s ?", name),
			"",
			StatusStyle.Render("[y] Yes    [n] No"),
		),
	)
}

func (m *Model) alertSetView() string {
	typeStr := "Drop"
	if m.alertIsRise {
		typeStr = "Rise"
	}
	code := ""
	if m.alertTarget != nil {
		code = m.alertTarget.Code
	}
	return DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			TitleStyle.Render("Set Alert"),
			"",
			fmt.Sprintf("  Fund: %s", code),
			fmt.Sprintf("  Type: %s  [t] toggle", typeStr),
			"",
			m.textInput.View(),
			"",
			StatusStyle.Render("Enter threshold %, [Enter] to confirm"),
		),
	)
}

func (m *Model) settingsView() string {
	var rows []string
	for i := 0; i < 4; i++ {
		label := m.settingsFieldLabel(i)
		value := m.settingsFieldValue(i)
		row := fmt.Sprintf("  %s: %s", label, value)
		if m.settingsEditing && i == m.settingsIdx {
			row = row + "\n  > " + m.settingsEditInput.View()
		} else if i == m.settingsIdx {
			row = CursorStyle.Render(row)
		}
		rows = append(rows, row)
	}
	return DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			TitleStyle.Render("Settings"),
			"",
			lipgloss.JoinVertical(lipgloss.Left, rows...),
			"",
			StatusStyle.Render("[j/k] navigate  [Enter] edit  [Esc] save & back"),
		),
	)
}

func (m *Model) detailView() string {
	var sb strings.Builder

	if m.detailAsset.Kind == model.AssetKindStock {
		sym := m.detailAsset.Market + m.detailAsset.Code
		header := TitleStyle.Render(fmt.Sprintf(" Stock Detail: %s ", sym))
		sb.WriteString(header)
		sb.WriteString("\n\n")

		if q, ok := m.stockQuotes[sym]; ok && q != nil {
			sb.WriteString(fmt.Sprintf("  Name:        %s\n", q.Name))
			priceStr := fmt.Sprintf("%.2f", q.Value)
			sb.WriteString(fmt.Sprintf("  Price:       %s\n", priceStr))
			sb.WriteString(fmt.Sprintf("  Prev Close:  %.2f\n", q.Previous))
			sb.WriteString(fmt.Sprintf("  Change:      %s\n", RenderChange(q.ChangePct)))
			sb.WriteString(fmt.Sprintf("  Updated:     %s\n", q.UpdateTime))
		} else {
			sb.WriteString(fmt.Sprintf("  Code: %s\n", m.detailAsset.Code))
			sb.WriteString("  No quote data available.\n")
		}
		sb.WriteString("\n")
		sb.WriteString(StatusStyle.Render("历史分析暂未实现"))
		sb.WriteString("\n\n")
		sb.WriteString(StatusStyle.Render("[Esc] back to dashboard"))
		return sb.String()
	}

	header := TitleStyle.Render(fmt.Sprintf(" Fund Detail: %s ", m.detailAsset.Code))
	if m.detailAsset.Name != "" {
		header += "  " + StatusStyle.Render(m.detailAsset.Name)
	}
	sb.WriteString(header)
	sb.WriteString("\n\n")

	if m.detailLoading {
		sb.WriteString(LoadingStyle.Render("  Loading history data..."))
		sb.WriteString("\n")
		return sb.String()
	}

	if m.err != nil {
		sb.WriteString(ErrorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		sb.WriteString("\n")
		return sb.String()
	}

	tr := m.detailTrend
	sb.WriteString(fmt.Sprintf("  Direction:   %s\n", colorizeDirection(tr.Direction)))
	sb.WriteString(fmt.Sprintf("  5-day change: %.2f%%\n", tr.Change5D))
	sma5 := analysis.Latest(tr.SMA5)
	sma20 := analysis.Latest(tr.SMA20)
	rsi14 := analysis.Latest(tr.RSI14)
	if !isNaN(sma5) {
		sb.WriteString(fmt.Sprintf("  SMA(5):     %.4f\n", sma5))
	}
	if !isNaN(sma20) {
		sb.WriteString(fmt.Sprintf("  SMA(20):    %.4f\n", sma20))
	}
	if !isNaN(rsi14) {
		rsiLabel := "neutral"
		if rsi14 > 70 {
			rsiLabel = "overbought"
		} else if rsi14 < 30 {
			rsiLabel = "oversold"
		}
		sb.WriteString(fmt.Sprintf("  RSI(14):    %.2f (%s)\n", rsi14, rsiLabel))
	}
	sb.WriteString("\n")

	sb.WriteString(HeaderStyle.Render(
		padRight("Date", 14) + padRight("NAV", 12) + "Change%",
	))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", 40))
	sb.WriteString("\n")

	show := m.detailSnapshots
	if len(show) > 10 {
		show = show[len(show)-10:]
	}
	for i := len(show) - 1; i >= 0; i-- {
		s := show[i]
		chgStr := RenderChange(s.DailyGrowthPct)
		sb.WriteString(padRight(s.Date, 14) + padRight(fmt.Sprintf("%.4f", s.UnitNAV), 12) + chgStr + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(StatusStyle.Render("[Esc] back to dashboard"))

	return sb.String()
}

func (m *Model) helpView() string {
	return DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			TitleStyle.Render("Help"),
			"",
			"  j/k or ↑/↓   Navigate asset rows",
			"  a             Add fund or stock (6-digit → fund, sh/sz+code → stock)",
			"  d             Delete selected asset",
			"  A             Set alert (fund only)",
			"  s             Open settings",
			"  Enter         View detail (fund analysis / stock quote)",
			"  r             Manual refresh",
			"  h             Show this help",
			"  q or Esc      Quit application",
			"",
			StatusStyle.Render("Press any key to return"),
		),
	)
}

func colorizeDirection(dir string) string {
	switch dir {
	case "up":
		return PositiveStyle.Render(dir)
	case "down":
		return NegativeStyle.Render(dir)
	default:
		return ZeroStyle.Render(dir)
	}
}

func isNaN(f float64) bool {
	return f != f
}

// ---- Internal helpers ----

// resolveAssetList builds an ordered slice of AssetRow matching m.assetList,
// combining real-time fund data and stock quotes.
func (m *Model) resolveAssetList() []AssetRow {
	var rows []AssetRow
	for _, a := range m.assetList {
		switch a.Kind {
		case model.AssetKindFund:
			if rt, ok := m.realtime[a.Code]; ok && rt != nil {
				name := rt.Name
				if name == "" {
					name = a.Name
				}
				rows = append(rows, AssetRow{
					Kind:       model.AssetKindFund,
					Code:       a.Code,
					Name:       name,
					Available:  rt.Available,
					Value:      rt.EstimatedNAV,
					Previous:   rt.PreviousNAV,
					ChangePct:  rt.DailyChangePct,
					UpdateTime: rt.UpdateTime,
				})
			} else {
				rows = append(rows, AssetRow{
					Kind:      model.AssetKindFund,
					Code:      a.Code,
					Name:      a.Name,
					Available: false,
				})
			}
		case model.AssetKindStock:
			sym := a.Market + a.Code
			if q, ok := m.stockQuotes[sym]; ok && q != nil {
				rows = append(rows, AssetRow{
					Kind:       model.AssetKindStock,
					Market:     a.Market,
					Code:       a.Code,
					Name:       or(q.Name, a.Name),
					Available:  q.Available,
					Value:      q.Value,
					Previous:   q.Previous,
					ChangePct:  q.ChangePct,
					UpdateTime: q.UpdateTime,
				})
			} else {
				rows = append(rows, AssetRow{
					Kind:      model.AssetKindStock,
					Market:    a.Market,
					Code:      a.Code,
					Name:      a.Name,
					Available: false,
				})
			}
		}
	}
	return rows
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
		nameMap := make(map[string]string, len(m.assetList))
		for _, a := range m.assetList {
			nameMap[a.Code] = a.Name
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

// fetchDataCmd fetches real-time fund data, stock quotes, AND historical NAV data for sparklines.
func (m *Model) fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		funds := m.fetcher.FetchAllRealTime(m.config.FundCodes)

		var stockQuotes map[string]*model.Quote
		if len(m.config.StockSymbols) > 0 {
			var err error
			stockQuotes, err = m.fetcher.FetchStockQuotes(m.config.StockSymbols)
			if err != nil {
				slog.Warn("fetch stock quotes failed", "error", err)
				stockQuotes = make(map[string]*model.Quote)
			}
		}

		// Fetch nav history for sparklines (last 30 days, oldest→newest).
		navHist := make(map[string][]float64)
		for _, code := range m.config.FundCodes {
			snaps, err := m.store.GetNavHistory(code, 30)
			if err != nil || len(snaps) == 0 {
				continue
			}
			values := make([]float64, len(snaps))
			for i, snap := range snaps {
				values[len(snaps)-1-i] = snap.DailyGrowthPct
			}
			// Append today's real-time change so the rightmost block reflects now.
			if fund, ok := funds[code]; ok && fund != nil && fund.Available {
				values = append(values, fund.DailyChangePct)
			}
			navHist[code] = values
		}

		return dataFetchedMsg{
			funds:       funds,
			stockQuotes: stockQuotes,
			navHistory:  navHist,
		}
	}
}

// ---- Modal entry helpers ----

func (m *Model) enterAddFund() (tea.Model, tea.Cmd) {
	m.mode = modeAddFund
	m.textInput = newTextInput()
	m.textInput.Placeholder = "000000 or sh600519"
	m.textInput.CharLimit = 20
	m.textInput.Width = 30
	m.textInput.Focus()
	return m, nil
}

func (m *Model) updateAddFund(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "enter":
		input := strings.TrimSpace(m.textInput.Value())
		if input == "" {
			return m, nil
		}

		// Parse stock:sh:600519 format
		if strings.HasPrefix(input, "stock:") {
			parts := strings.SplitN(input[6:], ":", 2)
			if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 6 {
				return m, nil
			}
			return m, m.fetchAddAssetCmd(model.AssetKindStock, parts[0], parts[1])
		}

		// Parse sh600519 or sz000001 format (market prefix + 6-digit code)
		if len(input) == 8 {
			prefix := input[:2]
			code := input[2:]
			if (prefix == "sh" || prefix == "sz") && len(code) == 6 && isAllDigits(code) {
				return m, m.fetchAddAssetCmd(model.AssetKindStock, prefix, code)
			}
		}

		// Pure 6-digit → fund
		if len(input) == 6 && isAllDigits(input) {
			return m, m.fetchAddAssetCmd(model.AssetKindFund, "", input)
		}

		return m, nil
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

func (m *Model) enterConfirmDelete() (tea.Model, tea.Cmd) {
	if len(m.assetList) == 0 {
		return m, nil
	}
	target := m.assetList[m.cursor]
	m.confirmTarget = &target
	m.mode = modeConfirmDelete
	return m, nil
}

func (m *Model) enterAlertSet() (tea.Model, tea.Cmd) {
	if len(m.assetList) == 0 {
		return m, nil
	}
	target := m.assetList[m.cursor]
	if target.Kind == model.AssetKindStock {
		m.err = fmt.Errorf("股票告警暂未实现")
		return m, nil
	}
	m.alertTarget = &target
	m.alertIsRise = false
	m.mode = modeAlertSet
	m.textInput = newTextInput()
	m.textInput.Placeholder = "e.g. 3.0"
	m.textInput.CharLimit = 10
	m.textInput.Width = 20
	m.textInput.Focus()
	return m, nil
}

func (m *Model) enterDetail() (tea.Model, tea.Cmd) {
	if len(m.assetList) == 0 {
		return m, nil
	}
	target := m.assetList[m.cursor]
	m.detailAsset = &target
	m.mode = modeDetail

	if target.Kind == model.AssetKindStock {
		m.detailLoading = false
		m.detailSnapshots = nil
		return m, nil
	}

	m.detailLoading = true
	m.detailSnapshots = nil
	return m, m.fetchDetailCmd(target.Code)
}

// ---- Settings helpers ----

func (m *Model) settingsFieldLabel(idx int) string {
	switch idx {
	case 0:
		return "Refresh Interval (sec)"
	case 1:
		return "Cache TTL (min)"
	case 2:
		return "Alert Cooldown (min)"
	case 3:
		return "Max Concurrent Requests"
	default:
		return ""
	}
}

func (m *Model) settingsFieldValue(idx int) string {
	switch idx {
	case 0:
		return fmt.Sprintf("%d", m.appConfig.Settings.RefreshIntervalSec)
	case 1:
		return fmt.Sprintf("%d", m.appConfig.Settings.CacheTTLMin)
	case 2:
		return fmt.Sprintf("%d", m.appConfig.Settings.AlertCooldownMin)
	case 3:
		return fmt.Sprintf("%d", m.appConfig.Settings.MaxConcurrentRequests)
	default:
		return ""
	}
}

func (m *Model) applySettingsValue(idx int, val int) {
	switch idx {
	case 0:
		m.appConfig.Settings.RefreshIntervalSec = val
	case 1:
		m.appConfig.Settings.CacheTTLMin = val
	case 2:
		m.appConfig.Settings.AlertCooldownMin = val
	case 3:
		m.appConfig.Settings.MaxConcurrentRequests = val
	}
}

// ---- Fetch commands ----

func (m *Model) fetchAddAssetCmd(kind model.AssetKind, market, code string) tea.Cmd {
	return func() tea.Msg {
		if kind == model.AssetKindFund {
			nameMap, err := m.fetcher.BuildFundNameMap()
			if err != nil {
				return assetAddedMsg{kind: kind, code: code, err: err}
			}
			name, ok := nameMap[code]
			if !ok {
				return assetAddedMsg{kind: kind, code: code, err: fmt.Errorf("fund code %s not found", code)}
			}
			if err := m.store.AddFundWithName(code, name, model.FundUnknown); err != nil {
				return assetAddedMsg{kind: kind, code: code, err: err}
			}
			if err := m.store.AddAssetSimple(model.AssetKindFund, "", code); err != nil {
				return assetAddedMsg{kind: kind, code: code, err: err}
			}
			_ = m.store.UpdateFundName(code, name)
			return assetAddedMsg{kind: kind, code: code, name: name}
		}

		// Stock: fetch quote to discover name, then add to store.
		sym := market + code
		quotes, err := m.fetcher.FetchStockQuotes([]string{sym})
		name := ""
		if err == nil && quotes[sym] != nil && quotes[sym].Name != "" {
			name = quotes[sym].Name
		}
		if err := m.store.AddAssetSimple(model.AssetKindStock, market, code); err != nil {
			return assetAddedMsg{kind: kind, market: market, code: code, name: name, err: err}
		}
		if name != "" {
			_ = m.store.AddAssetWithName(model.AssetKindStock, market, code, name, 0)
		}
		return assetAddedMsg{kind: kind, market: market, code: code, name: name}
	}
}

func (m *Model) fetchDetailCmd(code string) tea.Cmd {
	return func() tea.Msg {
		snaps, err := m.store.GetNavHistory(code, 60)
		if err != nil {
			return detailFetchedMsg{err: err}
		}
		if len(snaps) < 5 {
			fetched, ferr := m.fetcher.FetchHistory(code, 60)
			if ferr != nil {
				return detailFetchedMsg{err: ferr}
			}
			_ = m.store.SaveNavSnapshots(fetched)
			snaps = fetched
		}
		if len(snaps) == 0 {
			return detailFetchedMsg{err: fmt.Errorf("no history for %s", code)}
		}
		var chrono []model.NavSnapshot
		for i := len(snaps) - 1; i >= 0; i-- {
			chrono = append(chrono, snaps[i])
		}
		trend := analysis.TrendSummary(chrono)
		return detailFetchedMsg{snapshots: chrono, trend: trend}
	}
}

// ---- Utility helpers ----

func parseFloatOrZero(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func removeFromSlice(slice []string, target string) []string {
	var result []string
	for _, s := range slice {
		if s != target {
			result = append(result, s)
		}
	}
	return result
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Compile-time interface check.
var _ tea.Model = (*Model)(nil)
