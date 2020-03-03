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
)

func initMetrics() {
	prometheus.MustRegister(EntryNotFoundCountMetric)
}
