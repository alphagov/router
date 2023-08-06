package router

import (
	"github.com/alphagov/router/handlers"
	"github.com/alphagov/router/triemux"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	internalServerErrorCountMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_internal_server_error_total",
			Help: "Number of 500 Internal Server Error responses originating from Router",
		},
		[]string{"host"},
	)

	routeReloadCountMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "router_route_reload_total",
			Help: "Total number of attempts to reload the routing table",
		},
	)

	routeReloadErrorCountMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "router_route_reload_error_total",
			Help: "Number of failed attempts to reload the routing table",
		},
	)

	routesCountMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "router_routes_loaded",
			Help: "Number of routes currently loaded",
		},
	)
)

func registerMetrics(r prometheus.Registerer) {
	r.MustRegister(
		internalServerErrorCountMetric,
		routeReloadCountMetric,
		routeReloadErrorCountMetric,
		routesCountMetric,
	)
	handlers.RegisterMetrics(r)
	triemux.RegisterMetrics(r)
}
