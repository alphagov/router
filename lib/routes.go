package router

import (
	"encoding/json"
)

const (
	HandlerTypeBackend  = "backend"
	HandlerTypeRedirect = "redirect"
	HandlerTypeGone     = "gone"
)

/*
IncomingPath is the URL path of the route (e.g. /foo)
RouteType is the type of matching the route should do (exact/prefix)
BackendID is the backend application (e.g. frontend, publisher etc...)
RedirectTo is the redirect location for a redirect route
SegmentsMode indicates whether the URL path for a redirect route should be preserved (preserve/ignore)
SchemaName indicates the type of route (backend, redirect, gone)
Details contains additional information about the route
*/
type Route struct {
	IncomingPath *string
	RouteType    *string
	BackendID    *string
	RedirectTo   *string
	SegmentsMode *string
	SchemaName   *string
	Details      *string
}

// Determine the handler type associated with a route
func (route *Route) handlerType() string {
	switch {
	case route.redirect():
		return HandlerTypeRedirect
	case route.gone():
		return HandlerTypeGone
	default:
		return HandlerTypeBackend
	}
}

/*
Returns the backend which should be used for backend routes. If the route
is a gone route and but has an non-empty details field (e.g. explanation)
then either route it to the backendID specified or default to frontend.
*/
func (route *Route) backend() *string {
	if route.SchemaName != nil && *route.SchemaName == "gone" && !route.gone() {
		if route.BackendID != nil {
			return route.BackendID
		} else {
			defaultBackend := "frontend"
			return &defaultBackend
		}
	}
	return route.BackendID
}

/*
Determine whether to return a gone handler for a gone route:
(i) Details field is nil
(ii) Details field isn't valid jSON

If the details field is empty (e.g. {}) then use a backend handler.
*/
func (route *Route) gone() bool {
	if route.SchemaName != nil && *route.SchemaName == "gone" {
		// If the details field is nil, use a standard gone route
		if route.Details == nil {
			return true
		}

		var detailsMap map[string]interface{}
		// If the details field is not valid JSON, use a standard gone route
		if err := json.Unmarshal([]byte(*route.Details), &detailsMap); err != nil {
			return true
		}

		// If the keys in the details map exist then return false
		for _, value := range detailsMap {
			// Not a standard gone route
			if value != nil && value != "" {
				return false
			}
		}
		// If the details field is empty, use a standard gone route
		return true
	}
	return false
}

func (route *Route) redirect() bool {
	return route.SchemaName != nil && *route.SchemaName == "redirect"
}

/*
Returns a flag (e.g. preserve, ignore) that is used to determine whether the URL path in a redirect route should be preserved.
Explicit logic to handle the case where a redirect route doesn't have a segmentsMode explicitly defined.
*/
func (route *Route) segmentsMode() string {
	if route.SegmentsMode == nil && route.SchemaName != nil && *route.SchemaName == "redirect" {
		if *route.RouteType == "prefix" {
			return "preserve"
		}
		return "ignore"
	}
	return *route.SegmentsMode
}
