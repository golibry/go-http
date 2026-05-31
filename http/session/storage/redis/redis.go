package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Storage provides session storage backed by Redis.
//
// Redis handles expiration natively through key TTLs, so Cleanup is a no-op.
type Storage struct {
	client goredis.UniversalClient
	prefix string
}

// New creates a new Redis-backed session storage.
func New(client goredis.UniversalClient, prefix string) *Storage {
	return &Storage{client: client, prefix: prefix}
}

// Get retrieves session data by ID. Returns (nil, nil) when not found.
func (rs *Storage) Get(ctx context.Context, sessionID string) ([]byte, error) {
	if sessionID == "" {
		return nil, nil
	}

	data, err := rs.client.Get(ctx, rs.key(sessionID)).Bytes()
	switch {
	case err == nil:
		return data, nil
	case errors.Is(err, goredis.Nil):
		return nil, nil
	default:
		return nil, err
	}
}

// Set stores session data with expiration TTL.
func (rs *Storage) Set(
	ctx context.Context,
	sessionID string,
	data []byte,
	expiration time.Duration,
) error {
	if sessionID == "" {
		return nil
	}

	return rs.client.Set(ctx, rs.key(sessionID), data, expiration).Err()
}

// Delete removes session data by ID.
func (rs *Storage) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	return rs.client.Del(ctx, rs.key(sessionID)).Err()
}

// Cleanup removes expired sessions. Redis expires session keys automatically.
func (rs *Storage) Cleanup(_ context.Context) error {
	return nil
}

// Exists checks if the session exists.
func (rs *Storage) Exists(ctx context.Context, sessionID string) bool {
	if sessionID == "" {
		return false
	}

	count, err := rs.client.Exists(ctx, rs.key(sessionID)).Result()
	return err == nil && count > 0
}

// Init validates the Redis connection.
func (rs *Storage) Init(ctx context.Context) error {
	if rs.client == nil {
		return errors.New("invalid storage configuration: redis client is nil")
	}

	return rs.client.Ping(ctx).Err()
}

func (rs *Storage) key(sessionID string) string {
	if rs.prefix == "" {
		return sessionID
	}

	return rs.prefix + sessionID
}
