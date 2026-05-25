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
