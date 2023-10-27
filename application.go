package luci

import (
	"net/http"
)

type Application interface {
	Routes() []Route
	Middlewares() []Middleware
	Error(http.ResponseWriter, *http.Request, int, error)
	Respond(http.ResponseWriter, *http.Request, any)
}
