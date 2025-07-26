package session

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// SessionTestSuite provides test suite for session functionality
type SessionTestSuite struct {
	suite.Suite
	storage Storage
	manager Manager
	ctx     context.Context
	logger  *slog.Logger
}

func (suite *SessionTestSuite) SetupTest() {
	suite.storage = NewMemoryStorage()
	suite.ctx = context.Background()
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	options := DefaultOptions()
	// Set encryption key for testing
	encryptionKey := make([]byte, 32)
	_, _ = rand.Read(encryptionKey)
	options.EncryptionKey = encryptionKey
	
	suite.manager = NewManager(suite.storage, suite.ctx, suite.logger, options)
}

func (suite *SessionTestSuite) TearDownTest() {
	if managerImpl, ok := suite.manager.(*ManagerImpl); ok {
		managerImpl.StopGC()
	}
}

func (suite *SessionTestSuite) TestItCanCreateNewSession() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	// Act
	session, err := suite.manager.NewSession(suite.ctx, w, r)

	// Assert
	suite.NoError(err)
	suite.NotNil(session)
	suite.NotEmpty(session.ID())
	
	// Check cookie was set
	cookies := w.Result().Cookies()
	suite.Len(cookies, 1)
	suite.Equal("session_id", cookies[0].Name)
	suite.Equal(session.ID(), cookies[0].Value)
}

func (suite *SessionTestSuite) TestItCanRetrieveExistingSession() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	
	// Create session first
	originalSession, err := suite.manager.NewSession(suite.ctx, w, r)
	suite.NoError(err)
	
	// Create new request with session cookie
	cookies := w.Result().Cookies()
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.AddCookie(cookies[0])

	// Act
	retrievedSession, err := suite.manager.GetSession(suite.ctx, r2)

	// Assert
	suite.NoError(err)
	suite.NotNil(retrievedSession)
	suite.Equal(originalSession.ID(), retrievedSession.ID())
}

func (suite *SessionTestSuite) TestItCanSetAndGetAttributes() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	session, err := suite.manager.NewSession(suite.ctx, w, r)
	suite.NoError(err)

	// Act
	session.Set("username", "testuser")
	session.Set("role", "admin")
	session.Set("count", 42)

	// Assert
	username, exists := session.Get("username")
	suite.True(exists)
	suite.Equal("testuser", username)
	
	role, exists := session.Get("role")
	suite.True(exists)
	suite.Equal("admin", role)
	
	count, exists := session.Get("count")
	suite.True(exists)
	suite.Equal(42, count)
	
	// Test non-existent key
	_, exists = session.Get("nonexistent")
	suite.False(exists)
}

func (suite *SessionTestSuite) TestItCanDeleteAttributes() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	session, err := suite.manager.NewSession(suite.ctx, w, r)
	suite.NoError(err)
	
	session.Set("key1", "value1")
	session.Set("key2", "value2")

	// Act
	session.Delete("key1")

	// Assert
	_, exists := session.Get("key1")
	suite.False(exists)
	
	value2, exists := session.Get("key2")
	suite.True(exists)
	suite.Equal("value2", value2)
}

func (suite *SessionTestSuite) TestItCanClearAllAttributes() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	session, err := suite.manager.NewSession(suite.ctx, w, r)
	suite.NoError(err)
	
	session.Set("key1", "value1")
	session.Set("key2", "value2")

	// Act
	session.Clear()

	// Assert
	_, exists := session.Get("key1")
	suite.False(exists)
	_, exists = session.Get("key2")
	suite.False(exists)
}

func (suite *SessionTestSuite) TestItCanAddAndGetFlashMessages() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	session, err := suite.manager.NewSession(suite.ctx, w, r)
	suite.NoError(err)

	// Act
	session.AddFlash("Success message")
	session.AddFlash("Error message", "error")
	session.AddFlash("Warning message", "warning")
	session.AddFlash("Another success", "success")

	// Assert
	defaultFlashes := session.GetFlashes()
	suite.Len(defaultFlashes, 1)
	suite.Equal("Success message", defaultFlashes[0])
	
	errorFlashes := session.GetFlashes("error")
	suite.Len(errorFlashes, 1)
	suite.Equal("Error message", errorFlashes[0])
	
	warningFlashes := session.GetFlashes("warning")
	suite.Len(warningFlashes, 1)
	suite.Equal("Warning message", warningFlashes[0])
	
	successFlashes := session.GetFlashes("success")
	suite.Len(successFlashes, 1)
	suite.Equal("Another success", successFlashes[0])
	
	// Flash messages should be consumed
	emptyFlashes := session.GetFlashes()
	suite.Len(emptyFlashes, 0)
}

func (suite *SessionTestSuite) TestItCanCheckSessionExpiration() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	session, err := suite.manager.NewSession(suite.ctx, w, r)
	suite.NoError(err)

	// Wait a bit to ensure time has passed
	time.Sleep(1 * time.Millisecond)

	// Act & Assert
	suite.False(session.IsExpired(time.Hour))
	suite.True(session.IsExpired(time.Nanosecond))
}

func (suite *SessionTestSuite) TestItCanTouchSession() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	session, err := suite.manager.NewSession(suite.ctx, w, r)
	suite.NoError(err)
	
	originalLastAccess := session.LastAccess()
	time.Sleep(10 * time.Millisecond)

	// Act
	session.Touch()

	// Assert
	newLastAccess := session.LastAccess()
	suite.True(newLastAccess.After(originalLastAccess))
}

func (suite *SessionTestSuite) TestItCanDestroySession() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	session, err := suite.manager.NewSession(suite.ctx, w, r)
	suite.NoError(err)
	
	sessionID := session.ID()
	session.Set("key", "value")

	// Act
	err = session.Destroy(suite.ctx)

	// Assert
	suite.NoError(err)
	
	// Session should not exist in storage
	suite.False(suite.storage.Exists(suite.ctx, sessionID))
	
	// Session data should be cleared
	_, exists := session.Get("key")
	suite.False(exists)
}

func (suite *SessionTestSuite) TestItCanSaveAndLoadSessionWithEncryption() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	session, err := suite.manager.NewSession(suite.ctx, w, r)
	suite.NoError(err)
	
	session.Set("encrypted_data", "sensitive information")
	session.AddFlash("encrypted flash")
	
	// Save session
	err = session.Save(suite.ctx)
	suite.NoError(err)
	
	// Create new request with session cookie
	cookies := w.Result().Cookies()
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.AddCookie(cookies[0])

	// Act
	loadedSession, err := suite.manager.GetSession(suite.ctx, r2)

	// Assert
	suite.NoError(err)
	suite.NotNil(loadedSession)
	
	value, exists := loadedSession.Get("encrypted_data")
	suite.True(exists)
	suite.Equal("sensitive information", value)
	
	flashes := loadedSession.GetFlashes()
	suite.Len(flashes, 1)
	suite.Equal("encrypted flash", flashes[0])
}

func (suite *SessionTestSuite) TestItCanStartAndStopGarbageCollection() {
	// Arrange
	managerImpl := suite.manager.(*ManagerImpl)

	// Act
	managerImpl.StartGC(suite.ctx)
	suite.True(managerImpl.gcRunning)
	
	managerImpl.StopGC()
	suite.False(managerImpl.gcRunning)
}

func (suite *SessionTestSuite) TestMemoryStorageCanCleanupExpiredSessions() {
	// Arrange
	memStorage := NewMemoryStorage()
	
	// Add expired session
	expiredData := []byte("expired")
	err := memStorage.Set(suite.ctx, "expired_session", expiredData, -time.Hour)
	suite.NoError(err)
	
	// Add valid session
	validData := []byte("valid")
	err = memStorage.Set(suite.ctx, "valid_session", validData, time.Hour)
	suite.NoError(err)

	// Act
	err = memStorage.Cleanup(suite.ctx)

	// Assert
	suite.NoError(err)
	suite.False(memStorage.Exists(suite.ctx, "expired_session"))
	suite.True(memStorage.Exists(suite.ctx, "valid_session"))
}

// MiddlewareTestSuite provides test suite for session middleware
type MiddlewareTestSuite struct {
	suite.Suite
	storage    Storage
	manager    Manager
	middleware *SessionMiddleware
	ctx        context.Context
	logger     *slog.Logger
}

func (suite *MiddlewareTestSuite) SetupTest() {
	suite.storage = NewMemoryStorage()
	suite.ctx = context.Background()
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	options := DefaultOptions()
	suite.manager = NewManager(suite.storage, suite.ctx, suite.logger, options)
	
	// Create a simple handler that uses session
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := GetSessionFromContext(r.Context())
		if ok && session != nil {
			session.Set("middleware_test", "success")
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	
	suite.middleware = NewSessionMiddleware(handler, suite.ctx, suite.logger, suite.manager)
}

func (suite *MiddlewareTestSuite) TestItCanHandleRequestWithoutSession() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	// Act
	suite.middleware.ServeHTTP(w, r)

	// Assert
	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *MiddlewareTestSuite) TestItCanHandleRequestWithExistingSession() {
	// Arrange
	// First create a session
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/", nil)
	session, err := suite.manager.NewSession(suite.ctx, w1, r1)
	suite.NoError(err)
	
	// Save the session explicitly
	err = session.Save(suite.ctx)
	suite.NoError(err)
	
	// Create new request with session cookie
	cookies := w1.Result().Cookies()
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.AddCookie(cookies[0])

	// Act
	suite.middleware.ServeHTTP(w2, r2)

	// Assert
	suite.Equal(http.StatusOK, w2.Code)
	
	// Get the session again to verify it was modified
	retrievedSession, err := suite.manager.GetSession(suite.ctx, r2)
	suite.NoError(err)
	
	// Verify session was modified
	value, exists := retrievedSession.Get("middleware_test")
	suite.True(exists)
	suite.Equal("success", value)
}

func (suite *MiddlewareTestSuite) TestItCanGetOrCreateSession() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	// Act
	session, err := GetOrCreateSession(suite.ctx, w, r, suite.manager)

	// Assert
	suite.NoError(err)
	suite.NotNil(session)
	suite.NotEmpty(session.ID())
}

// Run test suites
func TestSessionSuite(t *testing.T) {
	suite.Run(t, new(SessionTestSuite))
}

func TestMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}

// Additional unit tests
func TestDefaultOptions(t *testing.T) {
	options := DefaultOptions()
	
	assert.Equal(t, "session_id", options.CookieName)
	assert.Equal(t, "/", options.CookiePath)
	assert.Equal(t, 24*time.Hour, options.MaxAge)
	assert.Equal(t, 30*time.Minute, options.IdleTimeout)
	assert.Equal(t, 5*time.Minute, options.GCInterval)
	assert.True(t, options.SecureRandom)
	assert.True(t, options.CookieHTTPOnly)
	assert.Equal(t, http.SameSiteLaxMode, options.CookieSameSite)
}

func TestSessionErrors(t *testing.T) {
	assert.Equal(t, "session not found", ErrSessionNotFound.Error())
	assert.Equal(t, "invalid session", ErrInvalidSession.Error())
	assert.Equal(t, "encryption failed", ErrEncryptionFailed.Error())
	assert.Equal(t, "decryption failed", ErrDecryptionFailed.Error())
}