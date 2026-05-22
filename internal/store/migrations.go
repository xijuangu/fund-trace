package store

import (
	"fmt"
	"strings"

	"fund-trace/internal/model"
)

func (s *Store) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS funds (
		code TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		type INTEGER NOT NULL DEFAULT 0,
		added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS nav_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		fund_code TEXT NOT NULL,
		date TEXT NOT NULL,
		unit_nav REAL NOT NULL,
		accumulated_nav REAL NOT NULL,
		daily_growth_pct REAL NOT NULL DEFAULT 0,
		recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(fund_code, date)
	);

	CREATE INDEX IF NOT EXISTS idx_nav_code_date ON nav_snapshots(fund_code, date);

	CREATE TABLE IF NOT EXISTS alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		fund_code TEXT NOT NULL,
		type INTEGER NOT NULL DEFAULT 0,
		threshold_pct REAL NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		last_triggered_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS daily_summary (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		fund_code TEXT NOT NULL,
		nav REAL NOT NULL,
		change_pct REAL NOT NULL DEFAULT 0,
		note TEXT NOT NULL DEFAULT '',
		UNIQUE(date, fund_code)
	);

	CREATE TABLE IF NOT EXISTS assets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		kind INTEGER NOT NULL DEFAULT 0,
		market TEXT NOT NULL DEFAULT '',
		code TEXT NOT NULL,
		name TEXT NOT NULL DEFAULT '',
		type INTEGER NOT NULL DEFAULT 0,
		added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(kind, market, code)
	);

	CREATE TABLE IF NOT EXISTS price_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		kind INTEGER NOT NULL,
		market TEXT NOT NULL DEFAULT '',
		code TEXT NOT NULL,
		date TEXT NOT NULL,
		open REAL NOT NULL DEFAULT 0,
		high REAL NOT NULL DEFAULT 0,
		low REAL NOT NULL DEFAULT 0,
		close REAL NOT NULL DEFAULT 0,
		volume REAL NOT NULL DEFAULT 0,
		amount REAL NOT NULL DEFAULT 0,
		change_pct REAL NOT NULL DEFAULT 0,
		recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(kind, market, code, date)
	);

	CREATE INDEX IF NOT EXISTS idx_price_asset_date
	ON price_snapshots(kind, market, code, date);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if _, err := s.db.Exec(`INSERT OR IGNORE INTO assets (kind, market, code, name, type, added_at)
		SELECT 0, '', code, name, type, added_at FROM funds
		WHERE code NOT IN (SELECT code FROM assets WHERE kind = 0 AND market = '')`); err != nil {
		return fmt.Errorf("migrate funds to assets: %w", err)
	}

	alterCols := []struct{ col, def string }{
		{"kind", "INTEGER NOT NULL DEFAULT 0"},
		{"market", "TEXT NOT NULL DEFAULT ''"},
		{"code", "TEXT NOT NULL DEFAULT ''"},
	}
	for _, ac := range alterCols {
		_, err := s.db.Exec(fmt.Sprintf("ALTER TABLE alerts ADD COLUMN %s %s", ac.col, ac.def))
		if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("migrate alerts add %s: %w", ac.col, err)
		}
	}

	if _, err := s.db.Exec("UPDATE alerts SET code = fund_code WHERE code = ''"); err != nil {
		return fmt.Errorf("migrate alerts backfill code: %w", err)
	}

	return nil
}

func (s *Store) AssetOverview() ([]model.Asset, map[string][]model.Alert, error) {
	assets, err := s.ListAssets()
	if err != nil {
		return nil, nil, err
	}
	alertMap := make(map[string][]model.Alert)
	for _, a := range assets {
		key := alertKey(a.Kind, a.Market, a.Code)
		alerts, err := s.GetAlertsByAsset(a.Kind, a.Market, a.Code)
		if err != nil {
			return nil, nil, err
		}
		if len(alerts) > 0 {
			alertMap[key] = alerts
		}
	}
	return assets, alertMap, nil
}

func alertKey(kind model.AssetKind, market, code string) string {
	if kind == model.AssetKindFund {
		return code
	}
	return market + code
}
