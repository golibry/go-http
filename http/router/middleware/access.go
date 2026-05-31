package middleware

import (
	httpInternal "github.com/golibry/go-http/http"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const AccessLogMessage = "HTTP Request"

// extractClientIP safely extracts the client IP from RemoteAddr, handling both IPv4 and IPv6
func extractClientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If SplitHostPort fails, return the original address
		// This handles cases where there's no port or malformed address
		return remoteAddr
	}
	return host
}

type HTTPAccessLogger struct {
	next    http.Handler
	logger  *slog.Logger
	options AccessLogOptions
}

type AccessLogOptions struct {
	LogClientIp bool
}

func NewHTTPAccessLogger(
	next http.Handler,
	logger *slog.Logger,
	options AccessLogOptions,
) *HTTPAccessLogger {
	return &HTTPAccessLogger{next, logger, options}
}

func (accessLogger *HTTPAccessLogger) ServeHTTP(rw http.ResponseWriter, rq *http.Request) {
	logResponseWriter := httpInternal.NewResponseWriter(rw)
	timeBeforeServe := time.Now()
	accessLogger.next.ServeHTTP(logResponseWriter, rq)
	duration := time.Since(timeBeforeServe)

	var entries []slog.Attr

	if accessLogger.options.LogClientIp {
		clientIP := extractClientIP(rq.RemoteAddr)
		entries = append(
			entries,
			slog.String("client_ip", clientIP),
		)
	}

	entries = append(
		entries, []slog.Attr{
			slog.String("method", rq.Method),
			slog.String("host", rq.Host),
			slog.String("path", rq.URL.Path),
			slog.String("query", rq.URL.RawQuery),
			slog.String("protocol", rq.Proto),
			slog.String("user_agent", rq.UserAgent()),
			slog.Int("status", logResponseWriter.StatusCode()),
			slog.Int("bytes", logResponseWriter.BytesWritten()),
			slog.Int64("duration_ms", duration.Milliseconds()),
		}...,
	)

	accessLogger.logger.LogAttrs(
		rq.Context(),
		slog.LevelInfo,
		AccessLogMessage,
		entries...,
	)
}
