package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"fund-trace/internal/analysis"
	"fund-trace/internal/config"
	"fund-trace/internal/model"
	"fund-trace/internal/notifier"

	tea "github.com/charmbracelet/bubbletea"
)

var errTestDetailFetch = errors.New("fetch kline sh:588790")

func stockSnapshots() []model.PriceSnapshot {
	var snaps []model.PriceSnapshot
	for i := 1; i <= 20; i++ {
		snaps = append(snaps, model.PriceSnapshot{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			Date:      time.Date(2026, 5, i, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
			Close:     100 + float64(i),
			ChangePct: 0.5,
		})
	}
	return snaps
}

func TestStockDetailViewRendersHistoryAnalysis(t *testing.T) {
	snaps := stockSnapshots()
	closes := make([]float64, len(snaps))
	for i, s := range snaps {
		closes[i] = s.Close
	}

	m := &Model{
		detailAsset: &model.Asset{Kind: model.AssetKindStock, Market: "sh", Code: "600519", Name: "贵州茅台"},
		stockQuotes: map[string]*model.Quote{
			"sh600519": {
				Kind:       model.AssetKindStock,
				Market:     "sh",
				Code:       "600519",
				Name:       "贵州茅台",
				Value:      121,
				Previous:   120,
				ChangePct:  0.83,
				UpdateTime: "15:00:00",
				Available:  true,
			},
		},
		detailPriceSnapshots: snaps,
		detailTrend:          analysis.TrendSummaryFromValues(closes),
	}

	view := m.detailView()
	if strings.Contains(view, "历史分析暂未实现") {
		t.Fatalf("stock detail should render history analysis, got: %s", view)
	}
	for _, want := range []string{"SMA(5)", "RSI(14)", "2026-05-20", "Close"} {
		if !strings.Contains(view, want) {
			t.Fatalf("stock detail missing %q in view: %s", want, view)
		}
	}
}

func TestHandleStockDetailFetchedStoresQuote(t *testing.T) {
	m := &Model{
		detailAsset: &model.Asset{Kind: model.AssetKindStock, Market: "sh", Code: "688435"},
		stockQuotes: make(map[string]*model.Quote),
	}

	_, _ = m.handleDetailFetched(detailFetchedMsg{
		kind:   model.AssetKindStock,
		market: "sh",
		code:   "688435",
		quote: &model.Quote{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "688435",
			Name:      "英方软件",
			Value:     41.59,
			Previous:  41.57,
			Available: true,
		},
		priceSnapshots: stockSnapshots(),
	})

	if m.stockQuotes["sh688435"] == nil {
		t.Fatal("expected detail quote to be stored in stockQuotes")
	}
	view := m.detailView()
	if strings.Contains(view, "No quote data available") {
		t.Fatalf("expected detail view to use fetched quote, got: %s", view)
	}
	if !strings.Contains(view, "英方软件") {
		t.Fatalf("expected stock name from fetched quote, got: %s", view)
	}
}

func TestEnterAlertSetAllowsStock(t *testing.T) {
	m := &Model{
		assetList: []model.Asset{{Kind: model.AssetKindStock, Market: "sh", Code: "600519"}},
		cursor:    0,
	}

	_, _ = m.enterAlertSet()

	if m.mode != modeAlertSet {
		t.Fatalf("expected stock alert to enter alert mode, got mode %d err %v", m.mode, m.err)
	}
	if m.alertTarget == nil || m.alertTarget.Kind != model.AssetKindStock {
		t.Fatalf("expected stock alert target, got %#v", m.alertTarget)
	}
}

func TestCheckAlertsIncludesStocks(t *testing.T) {
	n := notifier.New(time.Minute)
	m := &Model{
		notifier: n,
		assetList: []model.Asset{
			{Kind: model.AssetKindStock, Market: "sh", Code: "600519", Name: "贵州茅台"},
		},
		stockQuotes: map[string]*model.Quote{
			"sh600519": {
				Kind:      model.AssetKindStock,
				Market:    "sh",
				Code:      "600519",
				Name:      "贵州茅台",
				ChangePct: -4.2,
				Available: true,
			},
		},
	}

	alerts := []model.Alert{{
		Kind:         model.AssetKindStock,
		Market:       "sh",
		Code:         "600519",
		Type:         model.AlertDrop,
		ThresholdPct: -3,
		Enabled:      true,
	}}

	triggered := m.checkStockAlerts(alerts)
	if len(triggered) != 1 {
		t.Fatalf("expected one triggered stock alert, got %d", len(triggered))
	}
}

func TestStockSymbolsForFetchUsesAssetListWhenConfigIsStale(t *testing.T) {
	m := &Model{
		config: Config{
			StockSymbols: []string{"sh600519"},
		},
		assetList: []model.Asset{
			{Kind: model.AssetKindStock, Market: "sh", Code: "688435", Name: "英方软件"},
		},
	}

	got := m.stockSymbolsForFetch()
	if len(got) != 1 || got[0] != "sh688435" {
		t.Fatalf("expected dashboard refresh to use stock assets from DB, got %#v", got)
	}
}

func TestSettingsIncludesColorSchemeToggle(t *testing.T) {
	m := &Model{
		appConfig: &config.Config{Settings: config.DefaultSettings()},
	}

	if got := m.settingsFieldLabel(3); got != "Change Color Scheme" {
		t.Fatalf("expected color scheme settings label, got %q", got)
	}
	if got := m.settingsFieldValue(3); got != "Green Up / Red Down" {
		t.Fatalf("expected default scheme label, got %q", got)
	}

	m.applySettingsValue(3, 0)
	if got := m.appConfig.Settings.ChangeColorScheme; got != "red_up_green_down" {
		t.Fatalf("expected toggled scheme red_up_green_down, got %q", got)
	}
	if got := m.settingsFieldValue(3); got != "Red Up / Green Down" {
		t.Fatalf("expected toggled scheme label, got %q", got)
	}
}

func TestSettingsEnterCommitsNumericEdit(t *testing.T) {
	input := newTextInput()
	input.SetValue("120")
	m := &Model{
		mode:              modeSettings,
		appConfig:         &config.Config{Settings: config.DefaultSettings()},
		settingsIdx:       0,
		settingsEditing:   true,
		settingsEditInput: input,
	}

	_, _ = m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})

	if m.settingsEditing {
		t.Fatal("expected enter to leave settings edit mode")
	}
	if got := m.appConfig.Settings.RefreshIntervalSec; got != 120 {
		t.Fatalf("expected refresh interval to be committed, got %d", got)
	}
}

func TestHandleDetailFetchedIgnoresStaleStockError(t *testing.T) {
	m := &Model{
		detailAsset:   &model.Asset{Kind: model.AssetKindStock, Market: "sh", Code: "512800"},
		detailLoading: true,
	}

	_, _ = m.handleDetailFetched(detailFetchedMsg{
		kind:   model.AssetKindStock,
		market: "sh",
		code:   "588790",
		err:    errTestDetailFetch,
	})

	if m.err != nil {
		t.Fatalf("stale detail error should be ignored, got %v", m.err)
	}
	if !m.detailLoading {
		t.Fatal("stale detail response should not stop current detail loading")
	}
}

func TestTickMsgReschedulesWhenNotNormal(t *testing.T) {
	m := &Model{
		mode: modeDetail,
		config: Config{
			RefreshInterval: time.Minute,
		},
	}

	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("tick should be rescheduled even when not in normal mode")
	}
}

func TestBuildStatusPartsKeepsNextRefreshAtZeroWhenOverdue(t *testing.T) {
	m := &Model{
		config: Config{
			RefreshInterval: time.Minute,
		},
		lastFetch: time.Now().Add(-2 * time.Minute),
	}

	parts := strings.Join(m.buildStatusParts(), " | ")
	if !strings.Contains(parts, "Next refresh: 0s") {
		t.Fatalf("expected overdue refresh to show 0s instead of disappearing, got %q", parts)
	}
}
