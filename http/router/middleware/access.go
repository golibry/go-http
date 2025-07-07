package middleware

import (
	"fmt"
	httpInternal "github.com/golibry/go-http/http"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const AccessLogMessage = "HTTP Request"

type HttpAccessLogger struct {
	nextHandler http.Handler
	logger      *slog.Logger
	options     AccessLogOptions
}

type AccessLogOptions struct {
	LogClientIp bool
}

func NewHttpAccessLogger(
	next http.Handler,
	logger *slog.Logger,
	options AccessLogOptions,
) *HttpAccessLogger {
	return &HttpAccessLogger{next, logger, options}
}

func (accessLogger *HttpAccessLogger) ServeHTTP(rw http.ResponseWriter, rq *http.Request) {
	logResponseWriter := httpInternal.NewResponseWriter(rw)
	timeBeforeServe := time.Now().UnixMilli()
	accessLogger.nextHandler.ServeHTTP(logResponseWriter, rq)
	timeAfterServe := time.Now().UnixMilli()

	var entries []slog.Attr

	if accessLogger.options.LogClientIp {
		entries = append(
			entries,
			slog.String("Client IP", strings.Split(rq.RemoteAddr, ":")[0]),
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
