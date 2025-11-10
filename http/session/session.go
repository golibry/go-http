package session

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Errors
var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrInvalidSession   = errors.New("invalid session")
	ErrEncryptionFailed = errors.New("encryption failed")
	ErrDecryptionFailed = errors.New("decryption failed")
)

// Storage interface for pluggable session backends
type Storage interface {
	// Get retrieves session data by ID
	Get(ctx context.Context, sessionID string) ([]byte, error)

	// Set stores session data with expiration
	Set(ctx context.Context, sessionID string, data []byte, expiration time.Duration) error

	// Delete removes session data
	Delete(ctx context.Context, sessionID string) error

	// Cleanup removes expired sessions
	Cleanup(ctx context.Context) error

	// Exists checks if session exists
	Exists(ctx context.Context, sessionID string) bool
}

// Session represents a user session
type Session interface {
	// ID returns the session ID
	ID() string

	// Get retrieves an attribute value
	Get(key string) (interface{}, bool)

	// Set stores an attribute value
	Set(key string, value interface{})

	// Delete removes an attribute
	Delete(key string)

	// Clear removes all attributes
	Clear()

	// AddFlash adds a flash message
	AddFlash(message interface{}, category ...string)

	// GetFlashes retrieves and removes flash messages
	GetFlashes(category ...string) []interface{}

	// Touch updates the last access time
	Touch()

	// LastAccess returns the last access time
	LastAccess() time.Time

	// CreatedAt returns the creation time
	CreatedAt() time.Time

	// IsExpired checks if session is expired
	IsExpired(maxAge time.Duration) bool

	// Save persists the session
	Save(ctx context.Context) error

	// Destroy removes the session
	Destroy(ctx context.Context) error
}

// Manager handles the session lifecycle
type Manager interface {
	// NewSession creates a new session
	NewSession(ctx context.Context, w http.ResponseWriter, r *http.Request) (Session, error)

	// GetSession retrieves existing session
	GetSession(ctx context.Context, r *http.Request) (Session, error)

	// DestroySession removes a session
	DestroySession(ctx context.Context, w http.ResponseWriter, r *http.Request) error

	// StartGC starts garbage collection
	StartGC(ctx context.Context)

	// StopGC stops garbage collection
	StopGC()
}

// SessionData holds the actual session data
type SessionData struct {
	ID         string                   `json:"id"`
	Attributes map[string]interface{}   `json:"attributes"`
	FlashData  map[string][]interface{} `json:"flash_data"`
	CreatedAt  time.Time                `json:"created_at"`
	LastAccess time.Time                `json:"last_access"`
}

// sessionImpl implements the Session interface
type sessionImpl struct {
	data    *SessionData
	storage Storage
	manager *ManagerImpl
	dirty   bool
	mu      sync.RWMutex
}

// ManagerImpl implements the Manager interface
type ManagerImpl struct {
	storage    Storage
	cookieName string
	options    Options
	gcTicker   *time.Ticker
	gcStop     chan struct{}
	gcRunning  bool
	mu         sync.RWMutex
	logger     *slog.Logger
	ctx        context.Context
}

// Options to configure session behavior
type Options struct {
	// Cookie settings
	CookieName     string
	CookiePath     string
	CookieDomain   string
	CookieSecure   bool
	CookieHTTPOnly bool
	CookieSameSite http.SameSite

	// Session settings
	MaxAge        time.Duration
	IdleTimeout   time.Duration
	EncryptionKey []byte // 32 bytes for AES-256

	// Garbage collection
	GCInterval time.Duration

	// Security
	SecureRandom bool
}

// DefaultOptions returns default session options
func DefaultOptions() Options {
	return Options{
		CookieName:     "session_id",
		CookiePath:     "/",
		CookieSecure:   false,
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
		MaxAge:         24 * time.Hour,
		IdleTimeout:    30 * time.Minute,
		GCInterval:     5 * time.Minute,
		SecureRandom:   true,
	}
}

// NewManager creates a new session manager
func NewManager(
	storage Storage,
	ctx context.Context,
	logger *slog.Logger,
	options Options,
) *ManagerImpl {
	return &ManagerImpl{
		storage:    storage,
		cookieName: options.CookieName,
		options:    options,
		gcStop:     make(chan struct{}),
		logger:     logger,
		ctx:        ctx,
	}
}

// generateSessionID creates a new session ID
func (m *ManagerImpl) generateSessionID() (string, error) {
	bytes := make([]byte, 64)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// NewSession creates a new session
func (m *ManagerImpl) NewSession(
	ctx context.Context,
	w http.ResponseWriter,
	_ *http.Request,
) (Session, error) {
	sessionID, err := m.generateSessionID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	data := &SessionData{
		ID:         sessionID,
		Attributes: make(map[string]interface{}),
		FlashData:  make(map[string][]interface{}),
		CreatedAt:  now,
		LastAccess: now,
	}

	session := &sessionImpl{
		data:    data,
		storage: m.storage,
		manager: m,
		dirty:   true,
	}

	// Set cookie
	cookie := &http.Cookie{
		Name:     m.options.CookieName,
		Value:    sessionID,
		Path:     m.options.CookiePath,
		Domain:   m.options.CookieDomain,
		MaxAge:   int(m.options.MaxAge.Seconds()),
		Secure:   m.options.CookieSecure,
		HttpOnly: m.options.CookieHTTPOnly,
		SameSite: m.options.CookieSameSite,
	}
	http.SetCookie(w, cookie)

	// Save session
	if err = session.Save(ctx); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves existing session
func (m *ManagerImpl) GetSession(ctx context.Context, r *http.Request) (Session, error) {
	cookie, err := r.Cookie(m.options.CookieName)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	sessionID := cookie.Value
	if sessionID == "" {
		return nil, ErrSessionNotFound
	}

	// Get session data from storage
	data, err := m.storage.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	} else if data == nil {
		return nil, ErrSessionNotFound
	}

	// Decrypt if encryption is enabled
	if len(m.options.EncryptionKey) > 0 {
		data, err = m.decrypt(data)
		if err != nil {
			return nil, ErrDecryptionFailed
		}
	}

	// Deserialize session data
	var sessionData SessionData
	if err = json.Unmarshal(data, &sessionData); err != nil {
		return nil, ErrInvalidSession
	}

	// Check if the session is expired
	session := &sessionImpl{
		data:    &sessionData,
		storage: m.storage,
		manager: m,
	}

	if session.IsExpired(m.options.MaxAge) || session.isIdleExpired(m.options.IdleTimeout) {
		_ = session.Destroy(ctx)
		return nil, ErrSessionNotFound
	}

	// Touch session to update last access time
	session.Touch()

	return session, nil
}

// DestroySession removes a session
func (m *ManagerImpl) DestroySession(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	session, err := m.GetSession(ctx, r)
	if err != nil {
		return err
	}

	// Remove cookie
	cookie := &http.Cookie{
		Name:     m.options.CookieName,
		Value:    "",
		Path:     m.options.CookiePath,
		Domain:   m.options.CookieDomain,
		MaxAge:   -1,
		Secure:   m.options.CookieSecure,
		HttpOnly: m.options.CookieHTTPOnly,
		SameSite: m.options.CookieSameSite,
	}
	http.SetCookie(w, cookie)

	return session.Destroy(ctx)
}

// StartGC starts garbage collection
func (m *ManagerImpl) StartGC(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.gcRunning {
		return
	}

	m.gcTicker = time.NewTicker(m.options.GCInterval)
	m.gcRunning = true

	go func() {
		for {
			select {
			case <-m.gcTicker.C:
				if err := m.storage.Cleanup(ctx); err != nil && m.logger != nil {
					m.logger.ErrorContext(ctx, "Session garbage collection failed", "error", err)
				}
			case <-m.gcStop:
				return
			}
		}
	}()
}

// StopGC stops garbage collection
func (m *ManagerImpl) StopGC() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.gcRunning {
		return
	}

	if m.gcTicker != nil {
		m.gcTicker.Stop()
	}
	close(m.gcStop)
	m.gcRunning = false
	m.gcStop = make(chan struct{})
}

// encrypt encrypts data using AES-GCM
func (m *ManagerImpl) encrypt(data []byte) ([]byte, error) {
	if len(m.options.EncryptionKey) == 0 {
		return data, nil
	}

	block, err := aes.NewCipher(m.options.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-GCM
func (m *ManagerImpl) decrypt(data []byte) ([]byte, error) {
	if len(m.options.EncryptionKey) == 0 {
		return data, nil
	}

	block, err := aes.NewCipher(m.options.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrDecryptionFailed
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// Session implementation methods

// ID returns the session ID
func (s *sessionImpl) ID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.ID
}

// Get retrieves an attribute value
func (s *sessionImpl) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.data.Attributes[key]
	return value, exists
}

// Set stores an attribute value
func (s *sessionImpl) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Attributes[key] = value
	s.dirty = true
}

// Delete removes an attribute
func (s *sessionImpl) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Attributes, key)
	s.dirty = true
}

// Clear removes all attributes
func (s *sessionImpl) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Attributes = make(map[string]interface{})
	s.dirty = true
}

// AddFlash adds a flash message
func (s *sessionImpl) AddFlash(message interface{}, category ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cat := "default"
	if len(category) > 0 && category[0] != "" {
		cat = category[0]
	}

	if s.data.FlashData == nil {
		s.data.FlashData = make(map[string][]interface{})
	}

	s.data.FlashData[cat] = append(s.data.FlashData[cat], message)
	s.dirty = true
}

// GetFlashes retrieves and removes flash messages
func (s *sessionImpl) GetFlashes(category ...string) []interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	cat := "default"
	if len(category) > 0 && category[0] != "" {
		cat = category[0]
	}

	if s.data.FlashData == nil {
		return nil
	}

	messages := s.data.FlashData[cat]
	delete(s.data.FlashData, cat)
	s.dirty = true

	return messages
}

// Touch updates the last access time
func (s *sessionImpl) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.LastAccess = time.Now()
	s.dirty = true
}

// LastAccess returns the last access time
func (s *sessionImpl) LastAccess() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.LastAccess
}

// CreatedAt returns the creation time
func (s *sessionImpl) CreatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.CreatedAt
}

// IsExpired checks if session is expired
func (s *sessionImpl) IsExpired(maxAge time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.data.CreatedAt) > maxAge
}

// isIdleExpired checks if the session is idle expired
func (s *sessionImpl) isIdleExpired(idleTimeout time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.data.LastAccess) > idleTimeout
}

// Save persists the session
func (s *sessionImpl) Save(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.dirty {
		return nil
	}

	// Serialize session data
	data, err := json.Marshal(s.data)
	if err != nil {
		return fmt.Errorf("failed to serialize session data: %w", err)
	}

	// Encrypt if encryption is enabled
	if len(s.manager.options.EncryptionKey) > 0 {
		data, err = s.manager.encrypt(data)
		if err != nil {
			return ErrEncryptionFailed
		}
	}

	// Store in storage
	if err := s.storage.Set(ctx, s.data.ID, data, s.manager.options.MaxAge); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	s.dirty = false
	return nil
}

// Destroy removes the session
func (s *sessionImpl) Destroy(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.storage.Delete(ctx, s.data.ID); err != nil {
		return fmt.Errorf("failed to destroy session: %w", err)
	}

	// Clear session data
	s.data.Attributes = make(map[string]interface{})
	s.data.FlashData = make(map[string][]interface{})
	s.dirty = false

	return nil
}
