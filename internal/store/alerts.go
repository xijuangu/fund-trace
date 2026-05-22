package store

import (
	"database/sql"
	"fmt"
	"fund-trace/internal/model"
)

func (s *Store) UpsertAlert(a model.Alert) (int64, error) {
	if a.Code == "" && a.FundCode != "" {
		a.Code = a.FundCode
	}
	var idArg any
	if a.ID != 0 {
		idArg = a.ID
	}
	result, err := s.db.Exec(
		`INSERT INTO alerts (id, kind, market, code, fund_code, type, threshold_pct, enabled, last_triggered_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			kind=excluded.kind,
			market=excluded.market,
			code=excluded.code,
			threshold_pct=excluded.threshold_pct,
			enabled=excluded.enabled`,
		idArg, int(a.Kind), a.Market, a.Code, a.FundCode, int(a.Type), a.ThresholdPct, boolToInt(a.Enabled), a.LastTriggeredAt,
	)
	if err != nil {
		return 0, fmt.Errorf("upsert alert: %w", err)
	}
	return result.LastInsertId()
}

func (s *Store) ListAlerts() ([]model.Alert, error) {
	rows, err := s.db.Query(
		"SELECT id, kind, market, code, fund_code, type, threshold_pct, enabled, last_triggered_at FROM alerts ORDER BY kind, market, code",
	)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	return scanAlerts(rows)
}

func (s *Store) DeleteAlert(id int64) error {
	_, err := s.db.Exec("DELETE FROM alerts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete alert %d: %w", id, err)
	}
	return nil
}

func (s *Store) DisableAlert(id int64) error {
	_, err := s.db.Exec("UPDATE alerts SET enabled = 0 WHERE id = ?", id)
	return err
}

func (s *Store) GetAlertsByAsset(kind model.AssetKind, market, code string) ([]model.Alert, error) {
	rows, err := s.db.Query(
		"SELECT id, kind, market, code, fund_code, type, threshold_pct, enabled, last_triggered_at FROM alerts WHERE kind = ? AND market = ? AND code = ? AND enabled = 1",
		int(kind), market, code,
	)
	if err != nil {
		return nil, fmt.Errorf("get alerts for %s:%s:%s: %w", kind, market, code, err)
	}
	defer rows.Close()

	return scanAlerts(rows)
}

func (s *Store) GetAlertsForFund(code string) ([]model.Alert, error) {
	return s.GetAlertsByAsset(model.AssetKindFund, "", code)
}

func (s *Store) UpdateAlertTriggeredAt(id int64) error {
	_, err := s.db.Exec("UPDATE alerts SET last_triggered_at = datetime('now') WHERE id = ?", id)
	return err
}

func scanAlerts(rows *sql.Rows) ([]model.Alert, error) {
	var alerts []model.Alert
	for rows.Next() {
		var a model.Alert
		var at int
		var enabled int
		var k int
		if err := rows.Scan(&a.ID, &k, &a.Market, &a.Code, &a.FundCode, &at, &a.ThresholdPct, &enabled, &a.LastTriggeredAt); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		a.Kind = model.AssetKind(k)
		a.Type = model.AlertType(at)
		a.Enabled = intToBool(enabled)
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}
