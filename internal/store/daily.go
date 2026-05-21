package store

import (
	"fmt"
	"fund-trace/internal/model"
)

func (s *Store) SaveDailySummary(sum model.DailySummary) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO daily_summary (date, fund_code, nav, change_pct, note) VALUES (?, ?, ?, ?, ?)",
		sum.Date, sum.FundCode, sum.NAV, sum.ChangePct, sum.Note,
	)
	if err != nil {
		return fmt.Errorf("save daily summary: %w", err)
	}
	return nil
}

func (s *Store) GetDailySummary(date, code string) (*model.DailySummary, error) {
	var sum model.DailySummary
	err := s.db.QueryRow(
		"SELECT date, fund_code, nav, change_pct, note FROM daily_summary WHERE date = ? AND fund_code = ?",
		date, code,
	).Scan(&sum.Date, &sum.FundCode, &sum.NAV, &sum.ChangePct, &sum.Note)
	if err != nil {
		return nil, fmt.Errorf("get daily summary: %w", err)
	}
	return &sum, nil
}
