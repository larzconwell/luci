package luci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDuration(t *testing.T) {
	t.Parallel()

	start := time.Now()
	<-time.After(50 * time.Millisecond)

	request := httptest.NewRequest(http.MethodGet, "/status", nil)
	request = request.WithContext(context.WithValue(request.Context(), durationKey{}, start))

	assert.GreaterOrEqual(t, Duration(request), 50*time.Millisecond)
}
