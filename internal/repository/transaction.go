package repository

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TransactionRecord represents a transaction record in database
type TransactionRecord struct {
	ID              int64     `json:"id"`
	TrxID           string    `json:"trx_id"`
	MessageID       string    `json:"message_id"`
	Destination     string    `json:"destination"`
	DestinationType string    `json:"destination_type"`
	SentAt          time.Time `json:"sent_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// TransactionRepository handles database operations for transactions
type TransactionRepository struct {
	db *sql.DB
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(dbPath string) (*TransactionRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trx_id TEXT NOT NULL UNIQUE,
			message_id TEXT NOT NULL,
			destination TEXT NOT NULL,
			destination_type TEXT NOT NULL,
			sent_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_trx_id ON transactions(trx_id);
		CREATE INDEX IF NOT EXISTS idx_expires_at ON transactions(expires_at);
		CREATE INDEX IF NOT EXISTS idx_destination ON transactions(destination);
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &TransactionRepository{db: db}, nil
}

// Close closes database connection
func (r *TransactionRepository) Close() error {
	return r.db.Close()
}

// Save saves a transaction record
func (r *TransactionRepository) Save(record *TransactionRecord) error {
	_, err := r.db.Exec(`
		INSERT INTO transactions (trx_id, message_id, destination, destination_type, sent_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, record.TrxID, record.MessageID, record.Destination, record.DestinationType, record.SentAt, record.ExpiresAt)
	return err
}

// GetByTrxID gets a transaction by TrxID (only non-expired)
func (r *TransactionRepository) GetByTrxID(trxID string) (*TransactionRecord, error) {
	var record TransactionRecord
	err := r.db.QueryRow(`
		SELECT id, trx_id, message_id, destination, destination_type, sent_at, expires_at, created_at
		FROM transactions
		WHERE trx_id = ? AND expires_at > ?
		LIMIT 1
	`, trxID, time.Now()).Scan(
		&record.ID,
		&record.TrxID,
		&record.MessageID,
		&record.Destination,
		&record.DestinationType,
		&record.SentAt,
		&record.ExpiresAt,
		&record.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByDestination gets transactions by destination (only non-expired)
func (r *TransactionRepository) GetByDestination(destination string) (*TransactionRecord, error) {
	var record TransactionRecord
	err := r.db.QueryRow(`
		SELECT id, trx_id, message_id, destination, destination_type, sent_at, expires_at, created_at
		FROM transactions
		WHERE destination = ? AND expires_at > ?
		ORDER BY sent_at DESC
		LIMIT 1
	`, destination, time.Now()).Scan(
		&record.ID,
		&record.TrxID,
		&record.MessageID,
		&record.Destination,
		&record.DestinationType,
		&record.SentAt,
		&record.ExpiresAt,
		&record.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// CleanupExpired removes expired transaction records
func (r *TransactionRepository) CleanupExpired() (int64, error) {
	result, err := r.db.Exec(`
		DELETE FROM transactions WHERE expires_at <= ?
	`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Count returns total active (non-expired) transactions
func (r *TransactionRepository) Count() (int64, error) {
	var count int64
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM transactions WHERE expires_at > ?
	`, time.Now()).Scan(&count)
	return count, err
}

