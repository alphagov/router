package triemux

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	entryNotFoundCountMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "router_triemux_entry_not_found_total",
			Help: "Number of route lookups for which no route was found",
		},
	)

	internalServiceUnavailableCountMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "router_service_unavailable_error_total",
			Help: "Number of 503 Service Unavailable errors originating from Router",
		},
	)
)

func RegisterMetrics(r prometheus.Registerer) {
	r.MustRegister(
		entryNotFoundCountMetric,
		internalServiceUnavailableCountMetric,
	)
}
