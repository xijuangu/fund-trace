package store

import (
	"fmt"
	"fund-trace/internal/model"
	"time"
)

func (s *Store) AddAssetSimple(kind model.AssetKind, market, code string) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO assets (kind, market, code, added_at) VALUES (?, ?, ?, ?)",
		int(kind), market, code, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("add asset %s:%s: %w", market, code, err)
	}
	return nil
}

func (s *Store) AddAssetWithName(kind model.AssetKind, market, code, name string, typ int) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO assets (kind, market, code, name, type, added_at) VALUES (?, ?, ?, ?, ?, ?)",
		int(kind), market, code, name, typ, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("add asset with name %s:%s: %w", market, code, err)
	}
	return nil
}

func (s *Store) RemoveAsset(kind model.AssetKind, market, code string) error {
	result, err := s.db.Exec(
		"DELETE FROM assets WHERE kind = ? AND market = ? AND code = ?",
		int(kind), market, code,
	)
	if err != nil {
		return fmt.Errorf("remove asset %s:%s: %w", market, code, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("asset %s:%s not found", market, code)
	}
	return nil
}

func (s *Store) ListAssets() ([]model.Asset, error) {
	rows, err := s.db.Query(
		"SELECT id, kind, market, code, name, type, added_at FROM assets ORDER BY kind, code",
	)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()

	var assets []model.Asset
	for rows.Next() {
		var a model.Asset
		var k int
		if err := rows.Scan(&a.ID, &k, &a.Market, &a.Code, &a.Name, &a.Type, &a.AddedAt); err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}
		a.Kind = model.AssetKind(k)
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

func (s *Store) GetAsset(kind model.AssetKind, market, code string) (*model.Asset, error) {
	var a model.Asset
	var k int
	err := s.db.QueryRow(
		"SELECT id, kind, market, code, name, type, added_at FROM assets WHERE kind = ? AND market = ? AND code = ?",
		int(kind), market, code,
	).Scan(&a.ID, &k, &a.Market, &a.Code, &a.Name, &a.Type, &a.AddedAt)
	if err != nil {
		return nil, fmt.Errorf("get asset %s:%s: %w", market, code, err)
	}
	a.Kind = model.AssetKind(k)
	return &a, nil
}

func (s *Store) UpdateAssetName(kind model.AssetKind, market, code, name string) error {
	_, err := s.db.Exec(
		"UPDATE assets SET name = ? WHERE kind = ? AND market = ? AND code = ? AND name = ''",
		name, int(kind), market, code,
	)
	return err
}
