package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
)

type Recoverer struct {
	next   http.Handler
	ctx    context.Context
	logger *slog.Logger
}

func (recoverer *Recoverer) ServeHTTP(rw http.ResponseWriter, rq *http.Request) {
	defer func() {
		if rvr := recover(); rvr != nil {
			var err error
			switch v := rvr.(type) {
			case string:
				err = errors.New(v)
			case error:
				err = v
			default:
				err = errors.New(fmt.Sprint(v))
			}

			if recoverer.logger != nil {
				recoverer.logger.ErrorContext(recoverer.ctx, err.Error())
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "Panic: %+v\n", rvr)
				debug.PrintStack()
			}

			http.Error(
				rw,
				http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError,
			)
		}
	}()

	recoverer.next.ServeHTTP(rw, rq)
}
