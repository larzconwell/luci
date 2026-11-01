package luci

import (
	"context"
	"net/http"
	"uuid"
)

type idKey struct{}

// ID returns the unique identifier associated with the request.
func ID(req *http.Request) string {
	id, _ := req.Context().Value(idKey{}).(string)
	return id
}

func withID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		xKey := "X-Request-Id"
		key := "Request-Id"
		xID := req.Header.Get(xKey)

		id := req.Header.Get(key)
		if id == "" {
			id = xID
		}

		if id == "" {
			id = uuid.NewV7().String()
		}

		header := rw.Header()
		header.Set(xKey, id)
		header.Set(key, id)

		newReq := req.WithContext(context.WithValue(req.Context(), idKey{}, id))
		next.ServeHTTP(rw, newReq)
	})
}
