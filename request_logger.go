package luci

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

type requestLoggerKey struct{}

// RequestLogger returns the logger associated with the request.
func RequestLogger(req *http.Request) *slog.Logger {
	logger, _ := req.Context().Value(requestLoggerKey{}).(*slog.Logger)
	return logger
}

func withRequestLogger(serverLogger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rww, ok := rw.(*responseWriterWrapper)
			if !ok {
				panic(errors.New("luci: response writer has not been wrapped"))
			}

			var varAttrs []any
			for key, value := range RequestVars(req) {
				varAttrs = append(varAttrs, slog.String(key, value))
			}

			logger := serverLogger.With(slog.Group(
				"request",
				slog.String("route", RequestRoute(req).Name),
				slog.String("id", ID(req)),
				slog.Group("vars", varAttrs...),
			))

			newReq := req.WithContext(context.WithValue(req.Context(), requestLoggerKey{}, logger))
			next.ServeHTTP(rww, newReq)

			logger.With(slog.Group(
				"response",
				slog.Int("status", rww.status),
				slog.Int("length", rww.length),
				slog.String("type", rww.Header().Get("Content-Type")),
			)).Info("Request")
		})
	}
}
