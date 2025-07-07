package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

// CustomHandler is like http.Handler but returns an error.
type CustomHandler func(w http.ResponseWriter, r *http.Request) error

type Errorhandler struct {
	next       CustomHandler
	ctx        context.Context
	logger     *slog.Logger
	clientErrs []error
}

func (handler *Errorhandler) ServeHTTP(rw http.ResponseWriter, rq *http.Request) {
	err := handler.next(rw, rq)
	if err != nil {
		if handler.logger != nil {
			handler.logger.ErrorContext(handler.ctx, err.Error())
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "Panic: %+v\n", err)
		}

		// Custom logic to distinguish client vs server error
		if errIsAny(err, handler.clientErrs) {
			http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		} else {
			http.Error(
				rw,
				http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError,
			)
		}
	}
}

func errIsAny(err error, target []error) bool {
	for _, t := range target {
		if errors.Is(err, t) {
			return true
		}
	}
	return false
}
