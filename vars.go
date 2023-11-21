package luci

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type varsKey struct{}

// Vars returns the request variables that are defined by the associated routes pattern.
func Vars(req *http.Request) map[string]string {
	vars, _ := req.Context().Value(varsKey{}).(map[string]string)
	return vars
}

// Var returns the request variable with the given key that is defined by the associated routes pattern.
func Var(req *http.Request, key string) string {
	return Vars(req)[key]
}

func withVars(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		urlParams := chi.RouteContext(req.Context()).URLParams

		vars := make(map[string]string, len(urlParams.Keys))
		for idx, key := range urlParams.Keys {
			vars[key] = urlParams.Values[idx]
		}

		newReq := req.WithContext(context.WithValue(req.Context(), varsKey{}, vars))
		next.ServeHTTP(rw, newReq)
	})
}
