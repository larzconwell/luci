package luci

import (
	"net/http"
)

// Application is used to define a server application and it's request/response behavior.
type Application interface {
	// Routes defines the routes an application supports.
	Routes() []Route
	// Middlewares defines the middlewares to run before any route specific middlewares.
	Middlewares() Middlewares
	// Error defines how the application responds to errors when handling a request.
	Error(rw http.ResponseWriter, req *http.Request, status int, err error)
	// Respond defines how the application reponds to requests that are successful.
	Respond(rw http.ResponseWriter, req *http.Request, value any)
}
