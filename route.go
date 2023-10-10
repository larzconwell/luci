package luci

import (
	"fmt"
	"net/http"
)

type routeContextKey struct{}

type Route struct {
	Name        string
	Method      string
	Pattern     string
	Middlewares []Middleware
	HandlerFunc http.HandlerFunc
}

func RequestRoute(req *http.Request) Route {
	route, _ := req.Context().Value(routeContextKey{}).(Route)
	return route
}

func (route Route) String() string {
	return fmt.Sprintf("%s %s %s", route.Name, route.Method, route.Pattern)
}
