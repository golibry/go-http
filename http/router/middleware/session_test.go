package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golibry/go-http/http/session"
	"github.com/stretchr/testify/suite"
)

type SessionMiddlewareTestSuite struct {
	suite.Suite
	storage    session.Storage
	manager    session.Manager
	middleware *SessionMiddleware
	ctx        context.Context
	logger     *slog.Logger
}

func TestSessionMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(SessionMiddlewareTestSuite))
}

func (suite *SessionMiddlewareTestSuite) SetupTest() {
	suite.storage = session.NewMemoryStorage()
	suite.ctx = context.Background()
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	options := session.DefaultOptions()
	suite.manager = session.NewManager(suite.storage, suite.ctx, suite.logger, options)

	// Create a simple handler that uses session
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			sess, ok := GetSessionFromContext(r.Context())
			if ok && sess != nil {
				sess.Set("middleware_test", "success")
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
	)

	suite.middleware = NewSessionMiddleware(handler, suite.ctx, suite.logger, suite.manager)
}

func (suite *SessionMiddlewareTestSuite) TestItCanHandleRequestWithoutSession() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	// Act
	suite.middleware.ServeHTTP(w, r)

	// Assert
	suite.Equal(http.StatusInternalServerError, w.Code)
}

func (suite *SessionMiddlewareTestSuite) TestItCanHandleRequestWithExistingSession() {
	// Arrange
	// First create a session
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/", nil)
	sess, err := suite.manager.NewSession(suite.ctx, w1, r1)
	suite.NoError(err)

	// Save the session explicitly
	err = sess.Save(suite.ctx)
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

func (suite *SessionMiddlewareTestSuite) TestItCanGetOrCreateSession() {
	// Arrange
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	// Act
	sess, err := GetOrCreateSession(suite.ctx, w, r, suite.manager)

	// Assert
	suite.NoError(err)
	suite.NotNil(sess)
	suite.NotEmpty(sess.ID())
}
