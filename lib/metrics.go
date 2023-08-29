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

	routeReloadDurationMetric = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "router_route_reload_duration_seconds",
			Help: "Histogram of route reload durations in seconds",
			Objectives: map[float64]float64{
				0.5:  0.01,
				0.9:  0.01,
				0.95: 0.01,
				0.99: 0.005,
			},
		},
		[]string{"success"},
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
		routeReloadDurationMetric,
		routesCountMetric,
	)
	handlers.RegisterMetrics(r)
	triemux.RegisterMetrics(r)
}
