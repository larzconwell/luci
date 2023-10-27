package luci

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type requestVarsKey struct{}

func RequestVars(req *http.Request) map[string]string {
	vars, _ := req.Context().Value(requestVarsKey{}).(map[string]string)
	return vars
}

func RequestVar(req *http.Request, key string) string {
	return RequestVars(req)[key]
}

func withRequestVars(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		urlParams := chi.RouteContext(req.Context()).URLParams

		vars := make(map[string]string, len(urlParams.Keys))
		for idx, key := range urlParams.Keys {
			vars[key] = urlParams.Values[idx]
		}

		newReq := req.WithContext(context.WithValue(req.Context(), requestVarsKey{}, vars))
		next.ServeHTTP(rw, newReq)
	})
}
