package luci

import (
	"errors"
	"net/http"
)

var (
	ErrMethodNotAllowed = errors.New("method not allowed")
	ErrNotFound         = errors.New("not found")
	ErrForcedShutdown   = errors.New("forced server shutdown")
)

type ErrorHandlerFunc func(http.ResponseWriter, *http.Request, int, error)

func errorRespond(errorHandler ErrorHandlerFunc, status int, err error) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		errorHandler(rw, req, status, err)
	}
}
