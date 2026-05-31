package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// PostgreSQLStorage provides session storage backed by PostgreSQL.
type PostgreSQLStorage struct {
	db        *sql.DB
	tableName string
}

// NewPostgreSQLStorage creates a new PostgreSQL-backed session storage.
// tableName can be a table name or a schema-qualified table name.
func NewPostgreSQLStorage(db *sql.DB, tableName string) *PostgreSQLStorage {
	return &PostgreSQLStorage{db: db, tableName: tableName}
}

// Get retrieves session data by ID. Returns (nil, nil) when not found or expired.
func (ps *PostgreSQLStorage) Get(ctx context.Context, sessionID string) ([]byte, error) {
	if sessionID == "" {
		return nil, nil
	}

	now := time.Now().UTC().Unix()
	query := "SELECT data FROM " + quotePostgreSQLTableName(ps.tableName) +
		" WHERE id = $1 AND expires_at > $2 LIMIT 1"
	row := ps.db.QueryRowContext(ctx, query, sessionID, now)

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
func (ps *PostgreSQLStorage) Set(
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

	stmt := "INSERT INTO " + quotePostgreSQLTableName(ps.tableName) +
		" (id, data, expires_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) " +
		"ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data, " +
		"expires_at = EXCLUDED.expires_at, updated_at = EXCLUDED.updated_at"
	_, err := ps.db.ExecContext(ctx, stmt, sessionID, data, expSec, nowSec, nowSec)
	return err
}

// Delete removes session data by ID.
func (ps *PostgreSQLStorage) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	stmt := "DELETE FROM " + quotePostgreSQLTableName(ps.tableName) + " WHERE id = $1"
	_, err := ps.db.ExecContext(ctx, stmt, sessionID)
	return err
}

// Cleanup removes expired sessions.
func (ps *PostgreSQLStorage) Cleanup(ctx context.Context) error {
	nowSec := time.Now().UTC().Unix()
	stmt := "DELETE FROM " + quotePostgreSQLTableName(ps.tableName) + " WHERE expires_at <= $1"
	_, err := ps.db.ExecContext(ctx, stmt, nowSec)
	return err
}

// Exists checks if the session exists and is not expired.
func (ps *PostgreSQLStorage) Exists(ctx context.Context, sessionID string) bool {
	if sessionID == "" {
		return false
	}

	nowSec := time.Now().UTC().Unix()
	query := "SELECT 1 FROM " + quotePostgreSQLTableName(ps.tableName) +
		" WHERE id = $1 AND expires_at > $2 LIMIT 1"
	row := ps.db.QueryRowContext(ctx, query, sessionID, nowSec)

	var one int
	if err := row.Scan(&one); err != nil {
		return false
	}

	return true
}

// Init creates the sessions table if it does not exist using BIGINT unix timestamps.
func (ps *PostgreSQLStorage) Init(ctx context.Context) error {
	if ps.db == nil || ps.tableName == "" {
		return errors.New("invalid storage configuration: db or table name is empty")
	}

	stmt := "CREATE TABLE IF NOT EXISTS " + quotePostgreSQLTableName(ps.tableName) + " (" +
		"id VARCHAR(191) PRIMARY KEY," +
		"data BYTEA NOT NULL," +
		"expires_at BIGINT NOT NULL," +
		"created_at BIGINT NOT NULL," +
		"updated_at BIGINT NOT NULL" +
		")"
	if _, err := ps.db.ExecContext(ctx, stmt); err != nil {
		return err
	}

	indexName := quotePostgreSQLIdentifier(ps.tableName + "_expires_at_idx")
	indexStmt := "CREATE INDEX IF NOT EXISTS " + indexName + " ON " +
		quotePostgreSQLTableName(ps.tableName) + " (expires_at)"
	_, err := ps.db.ExecContext(ctx, indexStmt)
	return err
}

func quotePostgreSQLTableName(tableName string) string {
	parts := strings.Split(tableName, ".")
	quotedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		quotedParts = append(quotedParts, quotePostgreSQLIdentifier(part))
	}

	return strings.Join(quotedParts, ".")
}

func quotePostgreSQLIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
