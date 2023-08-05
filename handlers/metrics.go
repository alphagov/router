package handlers

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	redirectHandlerRedirectCountMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_redirect_handler_redirect_total",
			Help: "Number of redirects handled by router redirect handlers",
		},
		[]string{
			"redirect_code",
			"redirect_type",
		},
	)

	backendHandlerRequestCountMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_backend_handler_request_total",
			Help: "Number of requests handled by router backend handlers",
		},
		[]string{
			"backend_id",
			"request_method",
		},
	)

	backendHandlerResponseDurationSecondsMetric = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "router_backend_handler_response_duration_seconds",
			Help: "Histogram of response durations by router backend handlers",
		},
		[]string{
			"backend_id",
			"request_method",
			"response_code",
		},
	)
)

func registerMetrics(r prometheus.Registerer) {
	r.MustRegister(
		backendHandlerRequestCountMetric,
		backendHandlerResponseDurationSecondsMetric,
		redirectHandlerRedirectCountMetric,
	)
}
