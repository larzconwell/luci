package luci

import (
	"context"
	"net/http"
	"time"
)

type durationKey struct{}

// Duration returns the duration of the request from the time it was started to the time Duration was called.
func Duration(req *http.Request) time.Duration {
	start, _ := req.Context().Value(durationKey{}).(time.Time)
	return time.Since(start)
}

func withDuration(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		start := time.Now()

		newReq := req.WithContext(context.WithValue(req.Context(), durationKey{}, start))
		next.ServeHTTP(rw, newReq)
	})
}
