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

	BackendHandlerRequestCountMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_backend_handler_request_count",
			Help: "Number of requests handled by router backend handlers",
		},
		[]string{
			"backend_id",
		},
	)

	BackendHandlerResponseDurationSecondsMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_backend_handler_response_duration_seconds",
			Help: "Time in seconds spent proxying requests to backends by router backend handlers",
		},
		[]string{
			"backend_id",
			"response_code",
		},
	)
)

func initMetrics() {
	prometheus.MustRegister(RedirectHandlerRedirectCountMetric)

	prometheus.MustRegister(BackendHandlerRequestCountMetric)
	prometheus.MustRegister(BackendHandlerResponseDurationSecondsMetric)
}
