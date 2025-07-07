package http

import (
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"testing"
)

type ResponseSuite struct {
	suite.Suite
}

func TestResponseSuite(t *testing.T) {
	suite.Run(t, new(ResponseSuite))
}

func (suite *ResponseSuite) TestResponseWriterCanCacheStatusCode() {
	expectedCode := 234
	baseResponseWriter := httptest.NewRecorder()
	responseWriter := NewResponseWriter(baseResponseWriter)
	responseWriter.WriteHeader(expectedCode)
	suite.Assert().Equal(expectedCode, baseResponseWriter.Code)
	suite.Assert().Equal(expectedCode, responseWriter.statusCode)
}
