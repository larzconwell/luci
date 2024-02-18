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

			requestAttrs := []any{slog.String("id", ID(req))}

			name := RequestRoute(req).Name
			if name != "" {
				requestAttrs = append(requestAttrs, slog.String("route", name))
			}

			var varAttrs []any
			for key, value := range Vars(req) {
				varAttrs = append(varAttrs, slog.String(key, value))
			}

			if len(varAttrs) > 0 {
				requestAttrs = append(requestAttrs, slog.Group("vars", varAttrs...))
			}

			logger := serverLogger.With(slog.Group(
				"request",
				requestAttrs...,
			))

			newReq := req.WithContext(context.WithValue(req.Context(), loggerKey{}, logger))
			next.ServeHTTP(wrw, newReq)

			_, status, length := wrw.stats()
			responseAttrs := []any{
				slog.String("duration", Duration(req).String()),
				slog.Int("status", status),
				slog.Int64("length", length),
			}

			contentType := wrw.Header().Get("Content-Type")
			if contentType != "" {
				responseAttrs = append(responseAttrs, slog.String("type", contentType))
			}

			logger.With(slog.Group(
				"response",
				responseAttrs...,
			)).Info("Request")
		})
	}
}
