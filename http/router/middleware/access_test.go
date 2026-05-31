package middleware

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/suite"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type AccessSuite struct {
	suite.Suite
}

func TestAccessSuite(t *testing.T) {
	suite.Run(t, new(AccessSuite))
}

type accessLog struct {
	Level      string `json:"level"`
	Msg        string `json:"msg"`
	IP         string `json:"client_ip"`
	Method     string `json:"method"`
	Host       string `json:"host"`
	Path       string `json:"path"`
	Query      string `json:"query"`
	Protocol   string `json:"protocol"`
	UserAgent  string `json:"user_agent"`
	Status     int    `json:"status"`
	Bytes      int    `json:"bytes"`
	DurationMS int64  `json:"duration_ms"`
}

func (suite *AccessSuite) TestItCanLogAccessDetails() {
	outputBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewJSONHandler(outputBuffer, &slog.HandlerOptions{}))

	expectedCode := 456
	expectedMethod := "GET"
	expectedPath := "/test123"
	expectedQuery := "test1=abx&test2=ert"
	expectedHost := "test234.com"
	expectedProtocol := "HTTP/1.1"
	expectedUserAgent := "Test User Agent 123"
	expectedIp := "123.456.789"
	request := httptest.NewRequest(
		expectedMethod, "https://"+expectedHost+expectedPath+"?"+expectedQuery,
		nil,
	)
	request.Header.Set("User-Agent", expectedUserAgent)
	request.RemoteAddr = expectedIp

	middleware := NewHTTPAccessLogger(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(expectedCode)
				_, _ = w.Write([]byte("hello"))
			},
		),
		logger,
		AccessLogOptions{true},
	)

	middleware.ServeHTTP(
		httptest.NewRecorder(),
		request,
	)

	loggedEntry := accessLog{}
	_ = json.Unmarshal(outputBuffer.Bytes(), &loggedEntry)

	suite.Assert().Equal(slog.LevelInfo.String(), loggedEntry.Level)
	suite.Assert().Equal(AccessLogMessage, loggedEntry.Msg)
	suite.Assert().Equal(expectedIp, loggedEntry.IP)
	suite.Assert().Equal(expectedMethod, loggedEntry.Method)
	suite.Assert().Equal(expectedHost, loggedEntry.Host)
	suite.Assert().Equal(expectedPath, loggedEntry.Path)
	suite.Assert().Equal(expectedQuery, loggedEntry.Query)
	suite.Assert().Equal(expectedProtocol, loggedEntry.Protocol)
	suite.Assert().Equal(expectedUserAgent, loggedEntry.UserAgent)
	suite.Assert().Equal(expectedCode, loggedEntry.Status)
	suite.Assert().Equal(5, loggedEntry.Bytes)
	suite.Assert().GreaterOrEqual(loggedEntry.DurationMS, int64(0))
}
