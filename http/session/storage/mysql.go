package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// MySQLStorage provides session storage backed by MySQL/MariaDB.
// It implements the session.Storage interface using a single table
// that stores the encrypted (or plain) blob of session data and an expiration time.
//
// This package does not include a MySQL driver. You must import and provide a
// configured *sql.DB (e.g., using github.com/go-sql-driver/mysql) in your app.
//
// Notes:
// - `expires_at` is managed by the library; cleanup will delete expired rows.
// - All times use unix epoch seconds in UTC; conversion is handled in the app.
// - The "191" limit for VARCHAR is safe for utf8mb4 primary keys in older MySQL versions.
//
// Usage:
//   db, _ := sql.Open("mysql", dsn)
//   store := storage.NewMySQLStorage(db, "sessions")
//   manager := session.NewManager(store, ctx, logger, options)
//
// the session manager handles The encryption (if any); this storage keeps bytes as-is.

type MySQLStorage struct {
	db        *sql.DB
	tableName string
}

// NewMySQLStorage creates a new MySQL/MariaDB-backed session storage.
// tableName should be the fully qualified table name (e.g., "sessions" or "schema.sessions").
func NewMySQLStorage(db *sql.DB, tableName string) *MySQLStorage {
	return &MySQLStorage{db: db, tableName: tableName}
}

// Get retrieves session data by ID. Returns (nil, nil) when not found or expired.
func (ms *MySQLStorage) Get(ctx context.Context, sessionID string) ([]byte, error) {
	if sessionID == "" {
		return nil, nil
	}

	// Only return non-expired sessions (expires_at is BIGINT unix seconds)
	now := time.Now().UTC().Unix()
	query := "SELECT data FROM " + ms.tableName + " WHERE id = ? AND expires_at > ? LIMIT 1"
	row := ms.db.QueryRowContext(ctx, query, sessionID, now)

	var data []byte
	switch err := row.Scan(&data); {
	case err == nil:
		return data, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	default:
		return nil, err
	}
}

// Set stores session data with expiration TTL. It upserts by ID.
func (ms *MySQLStorage) Set(
	ctx context.Context,
	sessionID string,
	data []byte,
	expiration time.Duration,
) error {
	if sessionID == "" {
		return nil
	}
	nowSec := time.Now().UTC().Unix()
	expSec := nowSec + int64(expiration.Seconds())

	// Use INSERT ... ON DUPLICATE KEY UPDATE for upsert
	stmt := "INSERT INTO " + ms.tableName + " (id, data, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?) " +
		"ON DUPLICATE KEY UPDATE data = VALUES(data), expires_at = VALUES(expires_at), updated_at = VALUES(updated_at)"
	_, err := ms.db.ExecContext(ctx, stmt, sessionID, data, expSec, nowSec, nowSec)
	return err
}

// Delete removes session data by ID.
func (ms *MySQLStorage) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	stmt := "DELETE FROM " + ms.tableName + " WHERE id = ?"
	_, err := ms.db.ExecContext(ctx, stmt, sessionID)
	return err
}

// Cleanup removes expired sessions.
func (ms *MySQLStorage) Cleanup(ctx context.Context) error {
	nowSec := time.Now().UTC().Unix()
	stmt := "DELETE FROM " + ms.tableName + " WHERE expires_at <= ?"
	_, err := ms.db.ExecContext(ctx, stmt, nowSec)
	return err
}

// Exists checks if the session exists and is not expired.
func (ms *MySQLStorage) Exists(ctx context.Context, sessionID string) bool {
	if sessionID == "" {
		return false
	}
	nowSec := time.Now().UTC().Unix()
	query := "SELECT 1 FROM " + ms.tableName + " WHERE id = ? AND expires_at > ? LIMIT 1"
	row := ms.db.QueryRowContext(ctx, query, sessionID, nowSec)
	var one int
	if err := row.Scan(&one); err != nil {
		return false
	}
	return true
}

// Init creates the sessions' table if it does not exist using BIGINT unix timestamps.
func (ms *MySQLStorage) Init(ctx context.Context) error {
	if ms.db == nil || ms.tableName == "" {
		return errors.New("invalid storage configuration: db or table name is empty")
	}
	stmt := "CREATE TABLE IF NOT EXISTS " + ms.tableName + " (" +
		"id VARCHAR(191) NOT NULL," +
		"data LONGBLOB NOT NULL," +
		"expires_at BIGINT NOT NULL," +
		"created_at BIGINT NOT NULL," +
		"updated_at BIGINT NOT NULL," +
		"PRIMARY KEY (id)," +
		"KEY idx_expires_at (expires_at)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci"
	_, err := ms.db.ExecContext(ctx, stmt)
	return err
}
