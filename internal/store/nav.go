package store

import (
	"fmt"
	"fund-trace/internal/model"
	"time"
)

func (s *Store) SaveNavSnapshots(snapshots []model.NavSnapshot) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		"INSERT OR IGNORE INTO nav_snapshots (fund_code, date, unit_nav, accumulated_nav, daily_growth_pct, recorded_at) VALUES (?, ?, ?, ?, ?, ?)",
	)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, snap := range snapshots {
		if _, err := stmt.Exec(snap.FundCode, snap.Date, snap.UnitNAV, snap.AccumulatedNAV, snap.DailyGrowthPct, now); err != nil {
			return fmt.Errorf("insert snapshot: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) GetNavHistory(code string, days int) ([]model.NavSnapshot, error) {
	rows, err := s.db.Query(
		"SELECT fund_code, date, unit_nav, accumulated_nav, daily_growth_pct, recorded_at FROM nav_snapshots WHERE fund_code = ? ORDER BY date DESC LIMIT ?",
		code, days,
	)
	if err != nil {
		return nil, fmt.Errorf("get nav history %s: %w", code, err)
	}
	defer rows.Close()

	var snaps []model.NavSnapshot
	for rows.Next() {
		var s model.NavSnapshot
		if err := rows.Scan(&s.FundCode, &s.Date, &s.UnitNAV, &s.AccumulatedNAV, &s.DailyGrowthPct, &s.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan nav: %w", err)
		}
		snaps = append(snaps, s)
	}
	return snaps, rows.Err()
}

func (s *Store) HasNavData(code string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM nav_snapshots WHERE fund_code = ?", code).Scan(&count)
	return count > 0, err
}
