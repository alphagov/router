package triemux

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	entryNotFoundCountMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "router_triemux_entry_not_found_total",
			Help: "Number of triemux lookups for which an entry was not found",
		},
	)

	internalServiceUnavailableCountMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "router_service_unavailable_error_total",
			Help: "Number of 503 Service Unavailable errors served by router",
		},
	)
)

func registerMetrics(r prometheus.Registerer) {
	r.MustRegister(
		entryNotFoundCountMetric,
		internalServiceUnavailableCountMetric,
	)
}
