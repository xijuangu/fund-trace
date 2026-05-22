package notifier

import (
	"testing"
	"time"

	"fund-trace/internal/model"
)

func TestCheckStockAlerts_DropTriggersWhenChangePctBelowThreshold(t *testing.T) {
	n := New(0)

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			ChangePct: -5.5,
			Available: true,
		},
	}

	alerts := []model.Alert{
		{
			ID:           1,
			Kind:         model.AssetKindStock,
			Market:       "sh",
			Code:         "600519",
			FundCode:     "600519",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 1 {
		t.Fatalf("expected 1 triggered alert, got %d", len(triggered))
	}
	if triggered[0].ID != 1 {
		t.Fatalf("expected alert ID 1, got %d", triggered[0].ID)
	}
}

func TestCheckStockAlerts_DropDoesNotTriggerWhenChangePctAboveThreshold(t *testing.T) {
	n := New(0)

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			ChangePct: -1.0,
			Available: true,
		},
	}

	alerts := []model.Alert{
		{
			ID:           1,
			Kind:         model.AssetKindStock,
			Market:       "sh",
			Code:         "600519",
			FundCode:     "600519",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 0 {
		t.Fatalf("expected 0 triggered alerts, got %d", len(triggered))
	}
}

func TestCheckStockAlerts_RiseTriggersWhenChangePctAboveThreshold(t *testing.T) {
	n := New(0)

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sz",
			Code:      "000001",
			ChangePct: 6.0,
			Available: true,
		},
	}

	alerts := []model.Alert{
		{
			ID:           2,
			Kind:         model.AssetKindStock,
			Market:       "sz",
			Code:         "000001",
			FundCode:     "000001",
			Type:         model.AlertRise,
			ThresholdPct: 5.0,
			Enabled:      true,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 1 {
		t.Fatalf("expected 1 triggered alert, got %d", len(triggered))
	}
	if triggered[0].ID != 2 {
		t.Fatalf("expected alert ID 2, got %d", triggered[0].ID)
	}
}

func TestCheckStockAlerts_RiseDoesNotTriggerWhenChangePctBelowThreshold(t *testing.T) {
	n := New(0)

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sz",
			Code:      "000001",
			ChangePct: 3.0,
			Available: true,
		},
	}

	alerts := []model.Alert{
		{
			ID:           2,
			Kind:         model.AssetKindStock,
			Market:       "sz",
			Code:         "000001",
			FundCode:     "000001",
			Type:         model.AlertRise,
			ThresholdPct: 5.0,
			Enabled:      true,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 0 {
		t.Fatalf("expected 0 triggered alerts, got %d", len(triggered))
	}
}

func TestCheckStockAlerts_DisabledAlertDoesNotTrigger(t *testing.T) {
	n := New(0)

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			ChangePct: -7.0,
			Available: true,
		},
	}

	alerts := []model.Alert{
		{
			ID:           1,
			Kind:         model.AssetKindStock,
			Market:       "sh",
			Code:         "600519",
			FundCode:     "600519",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      false,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 0 {
		t.Fatalf("expected 0 triggered alerts for disabled alert, got %d", len(triggered))
	}
}

func TestCheckStockAlerts_CooldownPreventsRepeatTrigger(t *testing.T) {
	n := New(1 * time.Hour)

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			ChangePct: -5.0,
			Available: true,
		},
	}

	alerts := []model.Alert{
		{
			ID:           1,
			Kind:         model.AssetKindStock,
			Market:       "sh",
			Code:         "600519",
			FundCode:     "600519",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 1 {
		t.Fatalf("first check: expected 1 triggered alert, got %d", len(triggered))
	}

	triggered = n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 0 {
		t.Fatalf("second check (cooldown): expected 0 triggered alerts, got %d", len(triggered))
	}
}

func TestCheckStockAlerts_CooldownResetsAfterDuration(t *testing.T) {
	n := New(10 * time.Millisecond)

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			ChangePct: -5.0,
			Available: true,
		},
	}

	alerts := []model.Alert{
		{
			ID:           1,
			Kind:         model.AssetKindStock,
			Market:       "sh",
			Code:         "600519",
			FundCode:     "600519",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 1 {
		t.Fatalf("first check: expected 1 triggered alert, got %d", len(triggered))
	}

	time.Sleep(15 * time.Millisecond)

	triggered = n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 1 {
		t.Fatalf("after cooldown expiry: expected 1 triggered alert, got %d", len(triggered))
	}
}

func TestCheckStockAlerts_NonMatchingAlertDoesNotTrigger(t *testing.T) {
	n := New(0)

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			ChangePct: -5.0,
			Available: true,
		},
	}

	alerts := []model.Alert{
		{
			ID:           1,
			Kind:         model.AssetKindStock,
			Market:       "sz",
			Code:         "000001",
			FundCode:     "000001",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
		{
			ID:           2,
			Kind:         model.AssetKindFund,
			Market:       "",
			Code:         "011513",
			FundCode:     "011513",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 0 {
		t.Fatalf("expected 0 triggered alerts for non-matching, got %d", len(triggered))
	}
}

func TestCheckStockAlerts_UnavailableQuoteDoesNotTrigger(t *testing.T) {
	n := New(0)

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			ChangePct: -5.0,
			Available: false,
		},
	}

	alerts := []model.Alert{
		{
			ID:           1,
			Kind:         model.AssetKindStock,
			Market:       "sh",
			Code:         "600519",
			FundCode:     "600519",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 0 {
		t.Fatalf("expected 0 triggered alerts for unavailable quote, got %d", len(triggered))
	}
}

func TestCheckStockAlerts_MultipleAlertsMultipleQuotes(t *testing.T) {
	n := New(0)

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			ChangePct: -5.0,
			Available: true,
		},
		{
			Kind:      model.AssetKindStock,
			Market:    "sz",
			Code:      "000001",
			ChangePct: 6.0,
			Available: true,
		},
	}

	alerts := []model.Alert{
		{
			ID:           1,
			Kind:         model.AssetKindStock,
			Market:       "sh",
			Code:         "600519",
			FundCode:     "600519",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
		{
			ID:           2,
			Kind:         model.AssetKindStock,
			Market:       "sz",
			Code:         "000001",
			FundCode:     "000001",
			Type:         model.AlertRise,
			ThresholdPct: 5.0,
			Enabled:      true,
		},
		{
			ID:           3,
			Kind:         model.AssetKindStock,
			Market:       "sz",
			Code:         "000001",
			FundCode:     "000001",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 2 {
		t.Fatalf("expected 2 triggered alerts, got %d", len(triggered))
	}
	foundIDs := map[int64]bool{}
	for _, a := range triggered {
		foundIDs[a.ID] = true
	}
	if !foundIDs[1] {
		t.Fatal("expected alert ID 1 to be triggered")
	}
	if !foundIDs[2] {
		t.Fatal("expected alert ID 2 to be triggered")
	}
	if foundIDs[3] {
		t.Fatal("alert ID 3 should not be triggered (rise alert not matching a drop)")
	}
}

func TestCheckStockAlerts_ThresholdExactlyEqualTriggers(t *testing.T) {
	n := New(0)

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			ChangePct: -3.0,
			Available: true,
		},
	}

	alerts := []model.Alert{
		{
			ID:           1,
			Kind:         model.AssetKindStock,
			Market:       "sh",
			Code:         "600519",
			FundCode:     "600519",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 1 {
		t.Fatalf("expected alert to trigger at exact threshold, got %d triggered", len(triggered))
	}
}

func TestCheckStockAlerts_NilNotifier(t *testing.T) {
	var n *Notifier

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			ChangePct: -5.0,
			Available: true,
		},
	}

	alerts := []model.Alert{
		{
			ID:           1,
			Kind:         model.AssetKindStock,
			Market:       "sh",
			Code:         "600519",
			FundCode:     "600519",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
	}

	triggered := n.CheckStockAlerts(quotes, alerts)
	if len(triggered) != 0 {
		t.Fatalf("nil notifier should return nil, got %d", len(triggered))
	}
}

func TestCheckStockAlerts_EmptyInputs(t *testing.T) {
	n := New(0)

	if triggered := n.CheckStockAlerts(nil, nil); len(triggered) != 0 {
		t.Fatalf("empty inputs should return nil, got %d", len(triggered))
	}

	alerts := []model.Alert{
		{
			ID:           1,
			Kind:         model.AssetKindStock,
			Market:       "sh",
			Code:         "600519",
			FundCode:     "600519",
			Type:         model.AlertDrop,
			ThresholdPct: -3.0,
			Enabled:      true,
		},
	}

	if triggered := n.CheckStockAlerts(nil, alerts); len(triggered) != 0 {
		t.Fatalf("nil quotes should return nil, got %d", len(triggered))
	}

	quotes := []model.Quote{
		{
			Kind:      model.AssetKindStock,
			Market:    "sh",
			Code:      "600519",
			ChangePct: -5.0,
			Available: true,
		},
	}

	if triggered := n.CheckStockAlerts(quotes, nil); len(triggered) != 0 {
		t.Fatalf("nil alerts should return nil, got %d", len(triggered))
	}
}
