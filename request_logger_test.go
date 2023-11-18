package luci

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestLogger(t *testing.T) {
	t.Parallel()

	logger := slog.Default()

	request := httptest.NewRequest(http.MethodGet, "/status", nil)
	request = request.WithContext(context.WithValue(request.Context(), requestLoggerKey{}, logger))

	assert.Same(t, logger, RequestLogger(request))
}
