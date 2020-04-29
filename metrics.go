package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	internalServerErrorCountMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_internal_server_error_total",
			Help: "Number of internal server errors encountered by router",
		},
		[]string{"host"},
	)

	routeReloadCountMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "router_route_reload_total",
			Help: "Number of times routing table has been reloaded",
		},
	)

	routeReloadErrorCountMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "router_route_reload_error_total",
			Help: "Number of errors encountered by reloading routing table",
		},
	)

	routesCountMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "router_routes_loaded",
			Help: "Number of routes currently loaded",
		},
	)
)

func initMetrics() {
	prometheus.MustRegister(internalServerErrorCountMetric)

	prometheus.MustRegister(routeReloadCountMetric)
	prometheus.MustRegister(routeReloadErrorCountMetric)

	prometheus.MustRegister(routesCountMetric)
}
