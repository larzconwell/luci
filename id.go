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

type idKey struct{}

// ID returns the unique identifier associated with the request.
func ID(req *http.Request) string {
	id, _ := req.Context().Value(idKey{}).(string)
	return id
}

func withID(errorHandler ErrorHandlerFunc) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			xKey := "X-Request-Id"
			key := "Request-Id"

			xID := req.Header.Get(xKey)
			id := req.Header.Get(key)
			if id == "" {
				id = xID
			}

			if id == "" {
				now := ulid.Now()
				ulid, err := ulid.New(now, entropy)
				if err != nil {
					errorHandler(rw, req, http.StatusInternalServerError, fmt.Errorf("luci: id generate: %w", err))
					return
				}

				id = ulid.String()
			}

			header := rw.Header()
			header.Set(xKey, id)
			header.Set(key, id)

			newReq := req.WithContext(context.WithValue(req.Context(), idKey{}, id))
			next.ServeHTTP(rw, newReq)
		})
	}
}
