package handlers

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	redirectCountMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_redirect_handler_redirect_total",
			Help: "Number of redirects served by redirect handlers",
		},
		[]string{
			"redirect_code",
			"redirect_type",
		},
	)

	backendRequestCountMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_backend_handler_request_total",
			Help: "Number of requests served by backend handlers",
		},
		[]string{
			"backend_id",
			"request_method",
		},
	)

	backendResponseDurationSecondsMetric = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "router_backend_handler_response_duration_seconds",
			Help: "Histogram of response durations by backend",
		},
		[]string{
			"backend_id",
			"request_method",
			"response_code",
		},
	)
)

func RegisterMetrics(r prometheus.Registerer) {
	r.MustRegister(
		backendRequestCountMetric,
		backendResponseDurationSecondsMetric,
		redirectCountMetric,
	)
}
