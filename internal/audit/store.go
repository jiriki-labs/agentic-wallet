package audit

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS transactions (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	ts         TEXT NOT NULL,
	agent      TEXT NOT NULL DEFAULT '',
	merchant   TEXT NOT NULL,
	amount     TEXT NOT NULL,
	token      TEXT NOT NULL,
	chain      TEXT NOT NULL,
	reason     TEXT NOT NULL DEFAULT '',
	decision   TEXT NOT NULL,
	tx_hash    TEXT NOT NULL DEFAULT '',
	order_id   TEXT NOT NULL DEFAULT '',
	status     TEXT NOT NULL,
	nonce_hex  TEXT NOT NULL DEFAULT '',
	expires_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_ts ON transactions(ts);
CREATE INDEX IF NOT EXISTS idx_merchant ON transactions(merchant);
CREATE INDEX IF NOT EXISTS idx_status ON transactions(status);
`

// Store is the SQLite-backed audit log.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at the given path.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("mkdir audit db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer; enforce at the pool level
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB for transaction use by the policy engine.
func (s *Store) DB() *sql.DB {
	return s.db
}
