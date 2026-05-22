package tui

import (
	"strings"
	"testing"
	"time"

	"fund-trace/internal/analysis"
	"fund-trace/internal/model"
	"fund-trace/internal/notifier"
)

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
