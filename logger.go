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
				panic(errors.New("luci: response writer has not been wrapped"))
			}

			var varAttrs []any
			for key, value := range Vars(req) {
				varAttrs = append(varAttrs, slog.String(key, value))
			}

			logger := serverLogger.With(slog.Group(
				"request",
				slog.String("route", RequestRoute(req).Name),
				slog.String("id", ID(req)),
				slog.Group("vars", varAttrs...),
			))

			newReq := req.WithContext(context.WithValue(req.Context(), loggerKey{}, logger))
			next.ServeHTTP(wrw, newReq)

			logger.With(slog.Group(
				"response",
				slog.String("duration", Duration(req).String()),
				slog.Int("status", wrw.status),
				slog.Int("length", wrw.length),
				slog.String("type", wrw.Header().Get("Content-Type")),
			)).Info("Request")
		})
	}
}
