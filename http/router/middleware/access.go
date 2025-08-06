package middleware

import (
	"fmt"
	httpInternal "github.com/golibry/go-http/http"
	"log/slog"
	"net"
	"net/http"
	"strconv"
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
	timeBeforeServe := time.Now().UnixMilli()
	accessLogger.next.ServeHTTP(logResponseWriter, rq)
	timeAfterServe := time.Now().UnixMilli()

	var entries []slog.Attr

	if accessLogger.options.LogClientIp {
		clientIP := extractClientIP(rq.RemoteAddr)
		entries = append(
			entries,
			slog.String("Client IP", clientIP),
		)
	}

	entries = append(
		entries, []slog.Attr{
			slog.String("Method", rq.Method),
			slog.String("Host", rq.Host),
			slog.String("Path", rq.URL.Path),
			slog.String("Query", rq.URL.RawQuery),
			slog.String("Protocol", rq.Proto),
			slog.String("User Agent", rq.UserAgent()),
			slog.String("Response Status Code", strconv.Itoa(logResponseWriter.StatusCode())),
			slog.String(
				"Duration (s)",
				fmt.Sprintf("%.2f", float64(timeAfterServe-timeBeforeServe)/1000),
			),
		}...,
	)

	accessLogger.logger.LogAttrs(
		rq.Context(),
		slog.LevelInfo,
		AccessLogMessage,
		entries...,
	)
}
