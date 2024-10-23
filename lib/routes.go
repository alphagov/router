package router

import (
	"encoding/json"
)

type CsRoute struct {
	IncomingPath *string
	RouteType    *string
	BackendID    *string
	RedirectTo   *string
	SegmentsMode *string
	SchemaName   *string
	Details      *string
}

func (route *CsRoute) backend() *string {
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

func (route *CsRoute) handlerType() string {
	if route.redirect() {
		return "redirect"
	} else if route.gone() {
		return "gone"
	} else {
		return "backend"
	}
}

func (route *CsRoute) gone() bool {
	if route.SchemaName != nil && *route.SchemaName == "gone" {
		if route.Details == nil {
			return true
		}

		var detailsMap map[string]interface{}
		if err := json.Unmarshal([]byte(*route.Details), &detailsMap); err != nil {
			return false
		}
		for _, value := range detailsMap {
			if value != nil && value != "" {
				return false
			}
		}
		return true
	}
	return false
}

func (route *CsRoute) redirect() bool {
	return route.SchemaName != nil && *route.SchemaName == "redirect"
}

func (route *CsRoute) segmentsMode() string {
	if route.SegmentsMode == nil && route.SchemaName != nil && *route.SchemaName == "redirect" {
		if *route.RouteType == "prefix" {
			return "preserve"
		}
		return "ignore"
	}
	return *route.SegmentsMode
}
