package luci

import (
	"net/http"
)

// Middleware is used to define custom middleware functionality.
type Middleware func(http.Handler) http.Handler
