package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)"+
		"&_pragma=foreign_keys(ON)"+
		"&_pragma=busy_timeout(5000)"+
		"&_pragma=synchronous(NORMAL)"+
		"&_pragma=temp_store(MEMORY)"+
		"&_pragma=cache_size(-20000)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &Store{db: db}, nil
}

func OpenMemory() (*Store, error) {
	return Open(":memory:")
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Close() error {
	return s.db.Close()
}
