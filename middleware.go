package luci

import (
	"context"
	"net/http"
)

// Middleware is used to define custom middleware functionality.
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
