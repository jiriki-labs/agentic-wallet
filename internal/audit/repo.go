package audit

import (
	"database/sql"
	"fmt"
	"time"
)

// Record represents one row in the transactions table.
type Record struct {
	ID        int64
	TS        time.Time
	Agent     string
	Merchant  string
	Amount    string
	Token     string
	Chain     string
	Reason    string
	Decision  string
	TxHash    string
	OrderID   string
	Status    string
	NonceHex  string
	ExpiresAt *time.Time
}

// InsertTx inserts a new record within an existing transaction tx.
// Use this when the caller manages the transaction (e.g., policy engine).
func InsertTx(tx *sql.Tx, r *Record) (int64, error) {
	expiresAt := ""
	if r.ExpiresAt != nil {
		expiresAt = r.ExpiresAt.UTC().Format(time.RFC3339)
	}
	res, err := tx.Exec(`INSERT INTO transactions
		(ts, agent, merchant, amount, token, chain, reason, decision, tx_hash, order_id, status, nonce_hex, expires_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.TS.UTC().Format(time.RFC3339),
		r.Agent, r.Merchant, r.Amount, r.Token, r.Chain, r.Reason,
		r.Decision, r.TxHash, r.OrderID, r.Status, r.NonceHex, expiresAt,
	)
	if err != nil {
		return 0, fmt.Errorf("insert audit record: %w", err)
	}
	return res.LastInsertId()
}

// Insert inserts a record directly (auto-transaction). Use for post-payment inserts.
func (s *Store) Insert(r *Record) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	id, err := InsertTx(tx, r)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	return id, tx.Commit()
}

// UpdateTxHash updates the tx_hash, status, and order_id for a record.
func (s *Store) UpdateTxHash(id int64, txHash, status, orderID string) error {
	_, err := s.db.Exec(
		`UPDATE transactions SET tx_hash=?, status=?, order_id=? WHERE id=?`,
		txHash, status, orderID, id,
	)
	return err
}

// ExpirePending marks pending rows past their expiresAt as 'expired'.
func (s *Store) ExpirePending() error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`UPDATE transactions SET status='expired', decision='expired'
		 WHERE status='confirm-pending' AND expires_at != '' AND expires_at < ?`, now,
	)
	return err
}

// DailySumTx returns the sum of amounts for the given UTC date that count
// toward the daily limit (paid, confirm-pending not yet expired, dry-run).
// Must be called within a BEGIN IMMEDIATE transaction.
func DailySumTx(tx *sql.Tx, todayUTC string, now time.Time) (float64, error) {
	var sum float64
	err := tx.QueryRow(`
		SELECT COALESCE(SUM(CAST(amount AS REAL)), 0)
		FROM transactions
		WHERE ts >= ?
		  AND ts < date(?, '+1 day')
		  AND (
		    status IN ('paid', 'dry-run')
		    OR (status = 'confirm-pending' AND (expires_at = '' OR expires_at > ?))
		  )
	`, todayUTC, todayUTC, now.UTC().Format(time.RFC3339)).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("daily sum query: %w", err)
	}
	return sum, nil
}

// List returns records with optional filters.
func (s *Store) List(merchant string, since time.Time, limit int) ([]*Record, error) {
	query := `SELECT id, ts, agent, merchant, amount, token, chain, reason, decision, tx_hash, order_id, status, nonce_hex, expires_at FROM transactions WHERE 1=1`
	args := []interface{}{}
	if merchant != "" {
		query += ` AND merchant = ?`
		args = append(args, merchant)
	}
	if !since.IsZero() {
		query += ` AND ts >= ?`
		args = append(args, since.UTC().Format(time.RFC3339))
	}
	query += ` ORDER BY id DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*Record
	for rows.Next() {
		r := &Record{}
		var tsStr, expiresAtStr string
		if err := rows.Scan(&r.ID, &tsStr, &r.Agent, &r.Merchant, &r.Amount,
			&r.Token, &r.Chain, &r.Reason, &r.Decision, &r.TxHash,
			&r.OrderID, &r.Status, &r.NonceHex, &expiresAtStr); err != nil {
			return nil, err
		}
		r.TS, _ = time.Parse(time.RFC3339, tsStr)
		if expiresAtStr != "" {
			t, _ := time.Parse(time.RFC3339, expiresAtStr)
			r.ExpiresAt = &t
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
