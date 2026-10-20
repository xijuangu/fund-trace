package tui

import (
	"strings"
	"testing"

	"fund-trace/internal/model"
)

func TestRenderAssetTableRendersStockTrend(t *testing.T) {
	rows := []AssetRow{{
		Kind:      model.AssetKindStock,
		Market:    "sh",
		Code:      "600519",
		Name:      "贵州茅台",
		Available: true,
		Value:     1290.20,
		ChangePct: -1.59,
	}}

	trends := map[string][]float64{
		model.QuoteKey(model.AssetKindStock, "sh", "600519"): {-1.2, 0.3, -0.8, 1.1, -1.59},
	}

	out := RenderAssetTable(rows, trends, -1, 0, 20, 120)
	if strings.Contains(out, "  —\n") {
		t.Fatalf("expected stock row to render sparkline trend, got:\n%s", out)
	}
	if !strings.ContainsAny(out, "▁▂▃▄▅▆▇█") {
		t.Fatalf("expected sparkline block in stock trend, got:\n%s", out)
	}
}

func TestChangeCellFlags(t *testing.T) {
	derived := changeCell(0.69, true, false)
	if !strings.Contains(derived, "~") || !strings.Contains(derived, "+0.69%") {
		t.Errorf("derived cell = %q, want a ~ prefix and +0.69%%", derived)
	}

	uncertain := changeCell(-1.27, false, true)
	if !strings.HasPrefix(uncertain, "-1.27%") || !strings.HasSuffix(uncertain, "?") {
		t.Errorf("uncertain cell = %q, want a ? suffix", uncertain)
	}

	if plain := changeCell(0, false, false); strings.ContainsAny(plain, "~?") {
		t.Errorf("plain cell = %q, want no flags", plain)
	}
}

func TestShortUpdateTime(t *testing.T) {
	cases := map[string]string{
		"2026-10-20 11:33:00": "11:33:00",
		"2026/05/22 16:08:22": "16:08:22",
		"15:00:00":            "15:00:00",
		"":                    "—",
	}
	for in, want := range cases {
		if got := shortUpdateTime(in); got != want {
			t.Errorf("shortUpdateTime(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderAssetTableShowsDerivedValueAndUpdateTime(t *testing.T) {
	rows := []AssetRow{{
		Kind:       model.AssetKindFund,
		Code:       "012969",
		Name:       "鹏华国证半导体芯片ETF联接A",
		Available:  true,
		Value:      1.1284,
		ChangePct:  0.69,
		UpdateTime: "2026-10-20 11:32:00",
		Derived:    true,
	}}

	out := RenderAssetTable(rows, nil, -1, 0, 20, 160)
	if !strings.Contains(out, "Updated") {
		t.Fatalf("expected an Updated column, got:\n%s", out)
	}
	if !strings.Contains(out, "11:32:00") {
		t.Fatalf("expected the update time in the row, got:\n%s", out)
	}
	if !strings.Contains(out, "~") {
		t.Fatalf("expected a derived marker on the change value, got:\n%s", out)
	}
}
