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

	var actual any
	middleware(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		actual = req.Context().Value("time")
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/status", nil))

	assert.Exactly(t, expected, actual)
}
