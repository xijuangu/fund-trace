package store

import "fmt"

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
	`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// Migrate existing funds to assets table (one-time).
	if _, err := s.db.Exec(`
		INSERT OR IGNORE INTO assets (kind, market, code, name, type, added_at)
		SELECT 0, '', code, name, type, added_at FROM funds
		WHERE code NOT IN (SELECT code FROM assets WHERE kind = 0 AND market = '')
	`); err != nil {
		return fmt.Errorf("migrate funds to assets: %w", err)
	}

	return nil
}
