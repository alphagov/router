package handlers

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	RedirectHandlerRedirectCountMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_redirect_handler_redirect_count",
			Help: "Number of redirects handled by router redirect handlers",
		},
		[]string{
			"redirect_code",
			"redirect_type",
			"redirect_url",
		},
	)
)

func initMetrics() {
	prometheus.MustRegister(RedirectHandlerRedirectCountMetric)
}
