package luci

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
)

func TestErrorRespond(t *testing.T) {
	t.Parallel()

	var mock mock.Mock
	errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
		mock.MethodCalled("handler", rw, req, status, err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/status", nil)
	status := http.StatusNotFound
	err := ErrNotFound
	mock.On("handler", recorder, request, status, err)

	handler := errorRespond(errorHandler, http.StatusNotFound, err)
	handler(recorder, request)

	mock.AssertExpectations(t)
}
