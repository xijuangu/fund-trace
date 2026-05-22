package store

import (
	"fmt"
	"time"

	"fund-trace/internal/model"
)

func (s *Store) SavePriceSnapshots(snapshots []model.PriceSnapshot) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		"INSERT OR IGNORE INTO price_snapshots (kind, market, code, date, open, high, low, close, volume, amount, change_pct, recorded_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
	)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, snap := range snapshots {
		if _, err := stmt.Exec(
			int(snap.Kind), snap.Market, snap.Code, snap.Date,
			snap.Open, snap.High, snap.Low, snap.Close,
			snap.Volume, snap.Amount, snap.ChangePct, now,
		); err != nil {
			return fmt.Errorf("insert price snapshot: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) GetPriceHistory(kind model.AssetKind, market, code string, days int) ([]model.PriceSnapshot, error) {
	rows, err := s.db.Query(
		"SELECT kind, market, code, date, open, high, low, close, volume, amount, change_pct, recorded_at FROM price_snapshots WHERE kind = ? AND market = ? AND code = ? ORDER BY date DESC LIMIT ?",
		int(kind), market, code, days,
	)
	if err != nil {
		return nil, fmt.Errorf("get price history: %w", err)
	}
	defer rows.Close()

	var snaps []model.PriceSnapshot
	for rows.Next() {
		var s model.PriceSnapshot
		var k int
		if err := rows.Scan(&k, &s.Market, &s.Code, &s.Date, &s.Open, &s.High, &s.Low, &s.Close, &s.Volume, &s.Amount, &s.ChangePct, &s.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan price: %w", err)
		}
		s.Kind = model.AssetKind(k)
		snaps = append(snaps, s)
	}
	return snaps, rows.Err()
}

func (s *Store) HasPriceData(kind model.AssetKind, market, code string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM price_snapshots WHERE kind = ? AND market = ? AND code = ?",
		int(kind), market, code,
	).Scan(&count)
	return count > 0, err
}
