package luci

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type timeoutResponseWriter struct {
	*responseWriter
	err error
}

func (rw *timeoutResponseWriter) Write(b []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.err != nil {
		return 0, rw.err
	}

	return rw.lockedWrite(b)
}

func (rw *timeoutResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.err != nil {
		return 0, rw.err
	}

	return rw.lockedReadFrom(r)
}

func (rw *timeoutResponseWriter) error(err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	rw.err = err
}

func withTimeout(errorHandler ErrorHandlerFunc, timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			ctx, cancel := context.WithTimeout(req.Context(), timeout)
			defer cancel()

			req = req.WithContext(ctx)

			wrw, ok := rw.(*responseWriter)
			if !ok {
				panic(errors.New("luci: withTimeout has not been called with responseWriter"))
			}
			trw := &timeoutResponseWriter{responseWriter: wrw}

			done := make(chan struct{})
			panicChan := make(chan any, 1)

			go func() {
				defer func() {
					val := recover()
					if val != nil {
						panicChan <- val
					}
				}()

				next.ServeHTTP(trw, req)
				close(done)
			}()

			select {
			case val := <-panicChan:
				err, ok := val.(error)
				if ok && errors.Is(err, http.ErrAbortHandler) {
					return
				}

				panic(val)
			case <-done:
				return
			case <-ctx.Done():
				err := ctx.Err()
				if errors.Is(err, context.DeadlineExceeded) {
					err = http.ErrHandlerTimeout
				}

				trw.error(err)

				wroteHeader, _, _ := trw.stats()
				if wroteHeader {
					Logger(req).With(slog.Any("error", err)).Error("Unable to write timeout error response, response already written")
					return
				}

				errorHandler(wrw, req, http.StatusServiceUnavailable, err)
			}
		})
	}
}
