package luci

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

type loggerKey struct{}

// Logger returns the logger associated with the request.
func Logger(req *http.Request) *slog.Logger {
	logger, _ := req.Context().Value(loggerKey{}).(*slog.Logger)
	return logger
}

func withLogger(serverLogger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			wrw, ok := rw.(*responseWriter)
			if !ok {
				panic(errors.New("luci: withLogger has not been called with responseWriter"))
			}

			requestAttrs := []slog.Attr{slog.String("id", ID(req))}

			name := RequestRoute(req).Name
			if name != "" {
				requestAttrs = append(requestAttrs, slog.String("route", name))
			}

			requestVars := Vars(req)
			if len(requestVars) > 0 {
				varAttrs := make([]slog.Attr, 0, len(requestVars))
				for key, value := range requestVars {
					varAttrs = append(varAttrs, slog.String(key, value))
				}

				requestAttrs = append(requestAttrs, slog.GroupAttrs("vars", varAttrs...))
			}

			logger := serverLogger.With(slog.GroupAttrs("request", requestAttrs...))

			newReq := req.WithContext(context.WithValue(req.Context(), loggerKey{}, logger))
			next.ServeHTTP(wrw, newReq)

			_, status, length := wrw.stats()
			responseAttrs := []slog.Attr{
				slog.String("duration", Duration(req).String()),
				slog.Int("status", status),
				slog.Int64("length", length),
			}

			contentType := wrw.Header().Get("Content-Type")
			if contentType != "" {
				responseAttrs = append(responseAttrs, slog.String("type", contentType))
			}

			logger.With(slog.GroupAttrs(
				"response",
				responseAttrs...,
			)).Info("request")
		})
	}
}
