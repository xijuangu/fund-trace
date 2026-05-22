package fetcher

import (
	"encoding/json"
	"fund-trace/internal/model"
	"testing"
)

func klineJSON(klines []string) []byte {
	resp := map[string]any{
		"data": map[string]any{
			"klines": klines,
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

func TestParseEastMoneyKLine_ValidData(t *testing.T) {
	raw := klineJSON([]string{
		"2026-05-20,1405.00,1410.00,1415.00,1400.00,100000,1412000000,,-0.50,,0.80",
		"2026-05-21,1410.00,1405.00,1415.00,1400.00,120000,1692000000,,0.35,,0.90",
		"2026-05-22,1405.00,1410.01,1415.00,1402.00,113928,1605265185,,-0.07,,0.87",
	})

	snapshots, err := ParseEastMoneyKLine(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(snapshots))
	}

	s := snapshots[0]
	if s.Kind != model.AssetKindStock {
		t.Errorf("expected AssetKindStock, got %d", s.Kind)
	}
	if s.Date != "2026-05-20" {
		t.Errorf("expected date 2026-05-20, got %s", s.Date)
	}
	if s.Open != 1405.00 {
		t.Errorf("expected open 1405.00, got %.2f", s.Open)
	}
	if s.Close != 1410.00 {
		t.Errorf("expected close 1410.00, got %.2f", s.Close)
	}
	if s.High != 1415.00 {
		t.Errorf("expected high 1415.00, got %.2f", s.High)
	}
	if s.Low != 1400.00 {
		t.Errorf("expected low 1400.00, got %.2f", s.Low)
	}
	if s.Volume != 100000 {
		t.Errorf("expected volume 100000, got %.0f", s.Volume)
	}
	if s.Amount != 1412000000 {
		t.Errorf("expected amount 1412000000, got %.0f", s.Amount)
	}
	if s.ChangePct != -0.50 {
		t.Errorf("expected change_pct -0.50, got %.2f", s.ChangePct)
	}
	if s.RecordedAt.IsZero() {
		t.Error("expected non-zero RecordedAt")
	}

	last := snapshots[2]
	if last.Date != "2026-05-22" {
		t.Errorf("expected date 2026-05-22, got %s", last.Date)
	}
	if last.Close != 1410.01 {
		t.Errorf("expected close 1410.01, got %.2f", last.Close)
	}
	if last.ChangePct != -0.07 {
		t.Errorf("expected change_pct -0.07, got %.2f", last.ChangePct)
	}
}

func TestParseEastMoneyKLine_EmptyResponse(t *testing.T) {
	raw := klineJSON(nil)
	snapshots, err := ParseEastMoneyKLine(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(snapshots))
	}

	raw2 := klineJSON([]string{})
	snapshots2, err := ParseEastMoneyKLine(raw2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots2) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(snapshots2))
	}
}

func TestParseEastMoneyKLine_MalformedJSON(t *testing.T) {
	_, err := ParseEastMoneyKLine([]byte(`not json`))
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestParseEastMoneyKLine_Malformed(t *testing.T) {
	raw := klineJSON([]string{
		"2026-05-20,1405.00,1410.00,1415.00,1400.00,100000,1412000000",              // only 7 fields, need 8+
		"2026-05-21,1410.00,1405.00,1415.00,1400.00,120000,1692000000,,-0.50,,0.90", // valid
	})

	snapshots, err := ParseEastMoneyKLine(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot (malformed skipped), got %d", len(snapshots))
	}
	if snapshots[0].Date != "2026-05-21" {
		t.Errorf("expected date 2026-05-21, got %s", snapshots[0].Date)
	}
	if snapshots[0].Open != 1410.00 {
		t.Errorf("expected open 1410.00, got %.2f", snapshots[0].Open)
	}
}

func TestParseEastMoneyKLine_EightFieldsDoesNotPanic(t *testing.T) {
	raw := klineJSON([]string{
		"2026-05-20,1405.00,1410.00,1415.00,1400.00,100000,1412000000,",
	})

	snapshots, err := ParseEastMoneyKLine(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots) != 0 {
		t.Fatalf("expected 8-field kline to be skipped, got %#v", snapshots)
	}
}

func TestParseEastMoneyKLine_AllMalformed(t *testing.T) {
	raw := klineJSON([]string{
		"2026-05-20",                 // only 1 field
		"2026-05-21,1410.00,1405.00", // only 3 fields
	})

	snapshots, err := ParseEastMoneyKLine(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(snapshots))
	}
}

func TestParseEastMoneyKLine_EmptyFields(t *testing.T) {
	raw := klineJSON([]string{
		"2026-05-20,,,,,,,,,,,",
		"2026-05-21,1410.00,1405.00,,,0,0,,,,", // fields 4-5 empty (high, low)
	})

	snapshots, err := ParseEastMoneyKLine(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
	if snapshots[0].Open != 0 {
		t.Errorf("expected 0 open for empty field, got %.2f", snapshots[0].Open)
	}
	if snapshots[0].Close != 0 {
		t.Errorf("expected 0 close for empty field, got %.2f", snapshots[0].Close)
	}
	if snapshots[1].Open != 1410.00 {
		t.Errorf("expected open 1410.00, got %.2f", snapshots[1].Open)
	}
	if snapshots[1].Close != 1405.00 {
		t.Errorf("expected close 1405.00, got %.2f", snapshots[1].Close)
	}
	if snapshots[1].High != 0 {
		t.Errorf("expected 0 for empty high, got %.2f", snapshots[1].High)
	}
}
