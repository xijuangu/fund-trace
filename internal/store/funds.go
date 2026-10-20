package store

import (
	"fmt"
	"fund-trace/internal/model"
	"time"
)

func (s *Store) AddFund(code string) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO funds (code, added_at) VALUES (?, ?)",
		code, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("add fund %s: %w", code, err)
	}
	return nil
}

func (s *Store) AddFundWithName(code, name string, fundType model.FundType) error {
	// Upsert instead of INSERT OR REPLACE so cached benchmark metadata
	// (index_code/index_secid/...) survives re-adding an existing fund.
	_, err := s.db.Exec(
		`INSERT INTO funds (code, name, type, added_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(code) DO UPDATE SET name = excluded.name, type = excluded.type`,
		code, name, int(fundType), time.Now(),
	)
	if err != nil {
		return fmt.Errorf("add fund with name %s: %w", code, err)
	}
	return nil
}

func (s *Store) RemoveFund(code string) error {
	result, err := s.db.Exec("DELETE FROM funds WHERE code = ?", code)
	if err != nil {
		return fmt.Errorf("remove fund %s: %w", code, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("fund %s not found", code)
	}
	return nil
}

func (s *Store) ListFunds() ([]model.Fund, error) {
	rows, err := s.db.Query(
		"SELECT code, name, type, added_at FROM funds ORDER BY code",
	)
	if err != nil {
		return nil, fmt.Errorf("list funds: %w", err)
	}
	defer rows.Close()

	var funds []model.Fund
	for rows.Next() {
		var f model.Fund
		var ft int
		if err := rows.Scan(&f.Code, &f.Name, &ft, &f.AddedAt); err != nil {
			return nil, fmt.Errorf("scan fund: %w", err)
		}
		f.Type = model.FundType(ft)
		funds = append(funds, f)
	}
	return funds, rows.Err()
}

func (s *Store) GetFund(code string) (*model.Fund, error) {
	var f model.Fund
	var ft int
	err := s.db.QueryRow(
		"SELECT code, name, type, added_at FROM funds WHERE code = ?", code,
	).Scan(&f.Code, &f.Name, &ft, &f.AddedAt)
	if err != nil {
		return nil, fmt.Errorf("get fund %s: %w", code, err)
	}
	f.Type = model.FundType(ft)
	return &f, nil
}

func (s *Store) SeedFromConfig(codes []string) error {
	for _, code := range codes {
		if err := s.AddFund(code); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpdateFundName(code, name string) error {
	_, err := s.db.Exec(
		"UPDATE funds SET name = ? WHERE code = ? AND name = ''",
		name, code,
	)
	return err
}

// FundMetas returns the cached benchmark metadata for every fund, keyed by code.
func (s *Store) FundMetas() (map[string]model.FundMeta, error) {
	rows, err := s.db.Query(
		"SELECT code, index_code, index_secid, index_weight, index_derive FROM funds",
	)
	if err != nil {
		return nil, fmt.Errorf("list fund metas: %w", err)
	}
	defer rows.Close()

	metas := make(map[string]model.FundMeta)
	for rows.Next() {
		var m model.FundMeta
		var derive int
		if err := rows.Scan(&m.Code, &m.IndexCode, &m.IndexSecID, &m.IndexWeight, &derive); err != nil {
			return nil, fmt.Errorf("scan fund meta: %w", err)
		}
		m.DeriveOK = derive != 0
		metas[m.Code] = m
	}
	return metas, rows.Err()
}

// SaveFundMeta caches a fund's benchmark metadata.
func (s *Store) SaveFundMeta(meta model.FundMeta) error {
	derive := 0
	if meta.DeriveOK {
		derive = 1
	}
	_, err := s.db.Exec(
		`UPDATE funds SET index_code = ?, index_secid = ?, index_weight = ?, index_derive = ?
		 WHERE code = ?`,
		meta.IndexCode, meta.IndexSecID, meta.IndexWeight, derive, meta.Code,
	)
	if err != nil {
		return fmt.Errorf("save fund meta %s: %w", meta.Code, err)
	}
	return nil
}
