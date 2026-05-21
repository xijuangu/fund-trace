package store

import (
	"fmt"
	"fund-trace/internal/model"
)

func (s *Store) UpsertAlert(a model.Alert) (int64, error) {
	var idArg any
	if a.ID != 0 {
		idArg = a.ID
	}
	result, err := s.db.Exec(
		`INSERT INTO alerts (id, fund_code, type, threshold_pct, enabled, last_triggered_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET threshold_pct=excluded.threshold_pct, enabled=excluded.enabled`,
		idArg, a.FundCode, int(a.Type), a.ThresholdPct, boolToInt(a.Enabled), a.LastTriggeredAt,
	)
	if err != nil {
		return 0, fmt.Errorf("upsert alert: %w", err)
	}
	return result.LastInsertId()
}

func (s *Store) ListAlerts() ([]model.Alert, error) {
	rows, err := s.db.Query(
		"SELECT id, fund_code, type, threshold_pct, enabled, last_triggered_at FROM alerts ORDER BY fund_code",
	)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	var alerts []model.Alert
	for rows.Next() {
		var a model.Alert
		var at int
		var enabled int
		if err := rows.Scan(&a.ID, &a.FundCode, &at, &a.ThresholdPct, &enabled, &a.LastTriggeredAt); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		a.Type = model.AlertType(at)
		a.Enabled = intToBool(enabled)
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
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

func (s *Store) GetAlertsForFund(code string) ([]model.Alert, error) {
	rows, err := s.db.Query(
		"SELECT id, fund_code, type, threshold_pct, enabled, last_triggered_at FROM alerts WHERE fund_code = ? AND enabled = 1",
		code,
	)
	if err != nil {
		return nil, fmt.Errorf("get alerts for %s: %w", code, err)
	}
	defer rows.Close()

	var alerts []model.Alert
	for rows.Next() {
		var a model.Alert
		var at int
		var enabled int
		if err := rows.Scan(&a.ID, &a.FundCode, &at, &a.ThresholdPct, &enabled, &a.LastTriggeredAt); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		a.Type = model.AlertType(at)
		a.Enabled = intToBool(enabled)
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (s *Store) UpdateAlertTriggeredAt(id int64) error {
	_, err := s.db.Exec("UPDATE alerts SET last_triggered_at = datetime('now') WHERE id = ?", id)
	return err
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
