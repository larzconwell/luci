package luci

import (
	"context"
	"net/http"
)

// Middleware defines functionality that runs before an endpoint handler.
type Middleware func(http.Handler) http.Handler

// WithValue is a middleware that adds the key/value to the request context.
func WithValue(key, value any) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			newReq := req.WithContext(context.WithValue(req.Context(), key, value))
			next.ServeHTTP(rw, newReq)
		})
	}
}

// Middlewares defines a list of ordered middleware to run before an endpoint handler.
type Middlewares []Middleware

// Handler creates a handler that runs the middlewares in the defined order
// and then calls the given handler.
func (middlewares Middlewares) Handler(next http.Handler) http.Handler {
	if len(middlewares) == 0 {
		return next
	}

	handler := middlewares[len(middlewares)-1](next)
	for i := len(middlewares) - 2; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}

// Handler creates a handler that runs the middlewares in the defined order
// and then calls the given handler function.
func (middlewares Middlewares) HandlerFunc(next http.HandlerFunc) http.Handler {
	return middlewares.Handler(next)
}
