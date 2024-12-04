package router

import (
	"encoding/json"
)

const (
	HandlerTypeBackend  = "backend"
	HandlerTypeRedirect = "redirect"
	HandlerTypeGone     = "gone"
)

type Route struct {
	IncomingPath *string
	RouteType    *string
	BackendID    *string
	RedirectTo   *string
	SegmentsMode *string
	SchemaName   *string
	Details      *string
}

// returns the backend which should be used for this route
// if the route is a gone route, but has an explaination in the details field,
// then route to the backend, or by default to government-frontend
func (route *Route) backend() *string {
	if route.SchemaName != nil && *route.SchemaName == "gone" && !route.gone() {
		if route.BackendID != nil {
			return route.BackendID
		} else {
			defaultBackend := "government-frontend"
			return &defaultBackend
		}
	}
	return route.BackendID
}

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

func (route *Route) gone() bool {
	if route.SchemaName != nil && *route.SchemaName == "gone" {
		if route.Details == nil {
			// if the details field is empty, use a standard gone route
			return true
		}

		// deserialise the details field, which should be JSON
		var detailsMap map[string]interface{}
		if err := json.Unmarshal([]byte(*route.Details), &detailsMap); err != nil {
			// if the details field is not valid JSON, use a standard gone route
			return true
		}
		// check if keys in the details map are not empty
		for _, value := range detailsMap {
			if value != nil && value != "" {
				// not a standard gone route
				return false
			}
		}
		// if the details field is empty, use a standard gone route
		return true
	}
	return false
}

func (route *Route) redirect() bool {
	return route.SchemaName != nil && *route.SchemaName == "redirect"
}

func (route *Route) segmentsMode() string {
	if route.SegmentsMode == nil && route.SchemaName != nil && *route.SchemaName == "redirect" {
		if *route.RouteType == "prefix" {
			return "preserve"
		}
		return "ignore"
	}
	return *route.SegmentsMode
}
