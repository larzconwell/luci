package luci

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	varMatcher = regexp.MustCompile("^{([^:]*):?(.*)}$")
)

type requestRouteKey struct{}

// Route defines an endpoint an application supports.
type Route struct {
	// Name is used to uniquely identify a route by name.
	Name string
	// Method may be optionally used to specify the method a route supports.
	// If not set the route will be used for all methods.
	Method string
	// Defines the pattern that the route should match on.
	// Refer to github.com/go-chi/chi/v5 for details on defining patterns.
	Pattern string
	// Middlewares define the route specific middlewares to run after the application middlewares.
	Middlewares Middlewares
	// HandlerFunc defines the handler function to call to handle the request.
	HandlerFunc http.HandlerFunc
}

// RequestRoute retrieves the route that's associated with the given request.
func RequestRoute(req *http.Request) Route {
	route, _ := req.Context().Value(requestRouteKey{}).(Route)
	return route
}

// String returns the routes name, method, and pattern.
func (route Route) String() string {
	return fmt.Sprintf("%s %s %s", route.Name, route.Method, route.Pattern)
}

// Path builds an absolute path from the routes pattern using the given variable values.
// Variable values must be in the order defined by the routes pattern, and are validated
// against the associated regex matcher if one exists.
func (route Route) Path(vals ...string) (string, error) {
	if route.Pattern == "" {
		return "", errors.New("luci: route pattern must not be empty")
	}
	if route.Pattern[0] != '/' {
		return "", errors.New("luci: route pattern must begin with /")
	}

	var valIndex int
	var varCount int
	var builder strings.Builder
	pattern := []byte(route.Pattern)[1:]
	parts := bytes.Split(pattern, []byte{'/'})

	for _, part := range parts {
		_ = builder.WriteByte('/')
		var value string

		for len(part) != 0 {
			start := bytes.IndexAny(part, "{*")
			if start == -1 {
				value = string(part)
				break
			}

			isWildcard := part[start] == '*'
			end := start

			var name string
			var matcherStr string
			if !isWildcard {
				end = bytes.IndexByte(part, '}')
				if end == -1 {
					return "", errors.New("luci: invalid route pattern")
				}

				matches := varMatcher.FindStringSubmatch(string(part[start : end+1]))
				if len(matches) >= 2 {
					name = matches[1]
				}
				if len(matches) == 3 && matches[2] != "" {
					matcherStr = matches[2]
				}
			}

			// If there's no vals left, keep count of vars so an
			// accurate count can be returned in the final error
			if len(vals)-1 < valIndex {
				valIndex++
				varCount++
				part = part[end+1:]
				continue
			}

			val := vals[valIndex]
			if !isWildcard && strings.Contains(val, "/") {
				return "", fmt.Errorf(`luci: value for variable "%s" must not contain /`, name)
			}

			if matcherStr != "" {
				matcher, err := regexp.Compile(matcherStr)
				if err != nil {
					return "", fmt.Errorf(`luci: variable "%s" must have valid regex: %w`, name, err)
				}

				if !matcher.MatchString(val) {
					return "", fmt.Errorf(`luci: value for variable "%s" does not match regex`, name)
				}
			}

			value += string(part[:start]) + val

			valIndex++
			varCount++
			part = part[end+1:]
		}

		split := strings.Split(value, "/")
		for idx, val := range split {
			split[idx] = url.PathEscape(val)
		}

		_, _ = builder.WriteString(strings.Join(split, "/"))
	}

	if varCount != len(vals) {
		return "", fmt.Errorf("luci: must provide the expected number of values (expected %d received %d)", varCount, len(vals))
	}

	return builder.String(), nil
}
