package triemux

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	EntryNotFoundCountMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "router_triemux_entry_not_found_total",
			Help: "Number of triemux lookups for which an entry was not found",
		},
	)

	InternalServiceUnavailableCountMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_service_unavailable_error_total",
			Help: "Number of 503 Service Unavailable errors served by router",
		},
		[]string{"temporary_child"},
	)
)

func initMetrics() {
	prometheus.MustRegister(EntryNotFoundCountMetric)
	prometheus.MustRegister(InternalServiceUnavailableCountMetric)
}
