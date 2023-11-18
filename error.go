package luci

import (
	"errors"
	"net/http"
)

var (
	// ErrMethodNotAllowed is used for requests with a method that's not allowed by a route.
	ErrMethodNotAllowed = errors.New("luci: method not allowed")
	// ErrNotFound is used for requests with a path that doesn't match any routes.
	ErrNotFound = errors.New("luci: not found")
	// ErrForcedShutdown is used when server shutdown takes longer than the configured timeout.
	ErrForcedShutdown = errors.New("luci: forced server shutdown")
)

// ErrorHandlerFunc is used to define functions that handle error specific responses.
type ErrorHandlerFunc func(http.ResponseWriter, *http.Request, int, error)

func errorRespond(errorHandler ErrorHandlerFunc, status int, err error) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		errorHandler(rw, req, status, err)
	}
}
