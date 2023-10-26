package luci

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RequestVars(req *http.Request) map[string]string {
	urlParams := chi.RouteContext(req.Context()).URLParams
	params := make(map[string]string, len(urlParams.Keys))

	for idx, key := range urlParams.Keys {
		params[key] = urlParams.Values[idx]
	}

	return params
}

func RequestVar(req *http.Request, key string) string {
	return RequestVars(req)[key]
}
