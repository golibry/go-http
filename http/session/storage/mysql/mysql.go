package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Storage provides session storage backed by MySQL/MariaDB.
type Storage struct {
	db        *sql.DB
	tableName string
}

// New creates a new MySQL/MariaDB-backed session storage.
// tableName should be the fully qualified table name, for example "sessions" or "schema.sessions".
func New(db *sql.DB, tableName string) *Storage {
	return &Storage{db: db, tableName: tableName}
}

// Get retrieves session data by ID. Returns (nil, nil) when not found or expired.
func (ms *Storage) Get(ctx context.Context, sessionID string) ([]byte, error) {
	if sessionID == "" {
		return nil, nil
	}

	now := time.Now().UTC().Unix()
	query := "SELECT `data` FROM `" + ms.tableName + "` WHERE `id` = ? AND `expires_at` > ? LIMIT 1"
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
func (ms *Storage) Set(
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

	stmt := "INSERT INTO `" + ms.tableName +
		"` (`id`, `data`, `expires_at`, `created_at`, `updated_at`) VALUES (?, ?, ?, ?, ?) " +
		"ON DUPLICATE KEY UPDATE `data` = VALUES(`data`), " +
		"`expires_at` = VALUES(`expires_at`), `updated_at` = VALUES(`updated_at`)"
	_, err := ms.db.ExecContext(ctx, stmt, sessionID, data, expSec, nowSec, nowSec)
	return err
}

// Delete removes session data by ID.
func (ms *Storage) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	stmt := "DELETE FROM `" + ms.tableName + "` WHERE `id` = ?"
	_, err := ms.db.ExecContext(ctx, stmt, sessionID)
	return err
}

// Cleanup removes expired sessions.
func (ms *Storage) Cleanup(ctx context.Context) error {
	nowSec := time.Now().UTC().Unix()
	stmt := "DELETE FROM `" + ms.tableName + "` WHERE `expires_at` <= ?"
	_, err := ms.db.ExecContext(ctx, stmt, nowSec)
	return err
}

// Exists checks if the session exists and is not expired.
func (ms *Storage) Exists(ctx context.Context, sessionID string) bool {
	if sessionID == "" {
		return false
	}

	nowSec := time.Now().UTC().Unix()
	query := "SELECT 1 FROM `" + ms.tableName + "` WHERE `id` = ? AND `expires_at` > ? LIMIT 1"
	row := ms.db.QueryRowContext(ctx, query, sessionID, nowSec)

	var one int
	if err := row.Scan(&one); err != nil {
		return false
	}

	return true
}

// Init creates the sessions table if it does not exist using BIGINT unix timestamps.
func (ms *Storage) Init(ctx context.Context) error {
	if ms.db == nil || ms.tableName == "" {
		return errors.New("invalid storage configuration: db or table name is empty")
	}

	stmt := "CREATE TABLE IF NOT EXISTS `" + ms.tableName + "` (" +
		"`id` VARCHAR(191) NOT NULL," +
		"`data` LONGBLOB NOT NULL," +
		"`expires_at` BIGINT NOT NULL," +
		"`created_at` BIGINT NOT NULL," +
		"`updated_at` BIGINT NOT NULL," +
		"PRIMARY KEY (`id`)," +
		"KEY idx_expires_at (`expires_at`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci"
	_, err := ms.db.ExecContext(ctx, stmt)
	return err
}
