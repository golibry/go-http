package middleware

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/suite"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

type AccessSuite struct {
	suite.Suite
}

func TestAccessSuite(t *testing.T) {
	suite.Run(t, new(AccessSuite))
}

type accessLog struct {
	Level     string `json:"level"`
	Msg       string `json:"msg"`
	Ip        string `json:"Client IP"`
	Method    string `json:"Method"`
	Host      string `json:"Host"`
	Path      string `json:"Path"`
	Query     string `json:"Query"`
	Protocol  string `json:"Protocol"`
	UserAgent string `json:"User Agent"`
	Code      string `json:"Response Status Code"`
}

func (suite *AccessSuite) TestAccessLoggerCanLogAccessDetails() {
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

	middleware := NewHttpAccessLogger(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(expectedCode)
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
	suite.Assert().Equal(expectedIp, loggedEntry.Ip)
	suite.Assert().Equal(expectedMethod, loggedEntry.Method)
	suite.Assert().Equal(expectedHost, loggedEntry.Host)
	suite.Assert().Equal(expectedPath, loggedEntry.Path)
	suite.Assert().Equal(expectedQuery, loggedEntry.Query)
	suite.Assert().Equal(expectedProtocol, loggedEntry.Protocol)
	suite.Assert().Equal(expectedUserAgent, loggedEntry.UserAgent)
	suite.Assert().Equal(strconv.Itoa(expectedCode), loggedEntry.Code)
}
