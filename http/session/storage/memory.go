package storage

import (
	"context"
	"sync"
	"time"
)

// MemoryStorage provides in-memory session storage
// It implements session.Storage
// NOTE: This storage is intended for testing and single-instance apps.
type MemoryStorage struct {
	sessions map[string]*memorySession
	mu       sync.RWMutex
}

type memorySession struct {
	data      []byte
	expiresAt time.Time
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		sessions: make(map[string]*memorySession),
	}
}

// Get retrieves session data by ID
func (ms *MemoryStorage) Get(_ context.Context, sessionID string) ([]byte, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	s, exists := ms.sessions[sessionID]
	if !exists {
		return nil, nil
	}

	now := time.Now()
	if now.After(s.expiresAt) {
		delete(ms.sessions, sessionID)
		return nil, nil
	}

	return s.data, nil
}

// Set stores session data with expiration
func (ms *MemoryStorage) Set(
	_ context.Context,
	sessionID string,
	data []byte,
	expiration time.Duration,
) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.sessions[sessionID] = &memorySession{
		data:      data,
		expiresAt: time.Now().Add(expiration),
	}
	return nil
}

// Delete removes session data
func (ms *MemoryStorage) Delete(_ context.Context, sessionID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.sessions, sessionID)
	return nil
}

// Cleanup removes expired sessions
func (ms *MemoryStorage) Cleanup(_ context.Context) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	for id, s := range ms.sessions {
		if now.After(s.expiresAt) {
			delete(ms.sessions, id)
		}
	}
	return nil
}

// Exists checks if the session exists (and not expired)
func (ms *MemoryStorage) Exists(_ context.Context, sessionID string) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	s, exists := ms.sessions[sessionID]
	if !exists {
		return false
	}
	return time.Now().Before(s.expiresAt)
}
