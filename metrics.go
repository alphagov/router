package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	internalServerErrorCountMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_internal_server_error_count",
			Help: "Number of internal server errors encountered by router",
		},
		[]string{"host"},
	)

	routeReloadCountMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "router_route_reload_count",
			Help: "Number of times routing table has been reloaded",
		},
	)

	routeReloadErrorCountMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "router_route_reload_error_count",
			Help: "Number of errors encountered by reloading routing table",
		},
	)
)

func initMetrics() {
	prometheus.MustRegister(internalServerErrorCountMetric)

	prometheus.MustRegister(routeReloadCountMetric)
	prometheus.MustRegister(routeReloadErrorCountMetric)
}
