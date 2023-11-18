package luci

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/oklog/ulid/v2"
)

var (
	reader  = bufio.NewReaderSize(rand.Reader, 10*10)
	entropy = &ulid.LockedMonotonicReader{
		MonotonicReader: ulid.Monotonic(reader, 0),
	}
)

type requestIDKey struct{}

// RequestID returns the unique identifier associated with the request.
func RequestID(req *http.Request) string {
	id, _ := req.Context().Value(requestIDKey{}).(string)
	return id
}

func withRequestID(errorHandler ErrorHandlerFunc) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			id := req.Header.Get("X-Request-Id")
			if id == "" {
				now := ulid.Now()
				ulid, err := ulid.New(now, entropy)
				if err != nil {
					errorHandler(rw, req, http.StatusInternalServerError, fmt.Errorf("request id generate: %w", err))
					return
				}

				id = ulid.String()
			}

			newReq := req.WithContext(context.WithValue(req.Context(), requestIDKey{}, id))
			next.ServeHTTP(rw, newReq)
		})
	}
}
