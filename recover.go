package luci

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

func withRecover(errorHandler ErrorHandlerFunc) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			defer func() {
				val := recover()
				if val == nil {
					return
				}

				var err error
				switch v := val.(type) {
				case error:
					err = v
				default:
					err = fmt.Errorf("%+v", v)
				}

				if errors.Is(err, http.ErrAbortHandler) {
					return
				}

				wrw, ok := rw.(*responseWriter)
				if ok {
					wroteHeader, _, _ := wrw.stats()
					if wroteHeader {
						Logger(req).With(slog.Any("error", err)).Error("Unable to write recovered error response, response already written")
						return
					}
				}

				errorHandler(rw, req, http.StatusInternalServerError, err)
			}()

			next.ServeHTTP(rw, req)
		})
	}
}
