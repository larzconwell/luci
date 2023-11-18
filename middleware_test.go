package luci

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithValue(t *testing.T) {
	t.Parallel()

	expected := time.Now().UTC()
	middleware := WithValue("time", expected)

	handler := middleware(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Exactly(t, expected, req.Context().Value("time"))
	}))

	handler.ServeHTTP(nil, httptest.NewRequest(http.MethodGet, "/status", nil))
}
