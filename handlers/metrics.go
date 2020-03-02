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
			"request_method",
		},
	)

	BackendHandlerResponseDurationSecondsMetric = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "router_backend_handler_response_duration_seconds",
			Help: "Histogram of response durations by router backend handlers",
			Buckets: prometheus.ExponentialBuckets(
				0.25, 2, 6, // This buckets [...0.25  0.5  1  2  4  8...]
			),
		},
		[]string{
			"backend_id",
			"request_method",
			"response_code",
		},
	)
)

func initMetrics() {
	prometheus.MustRegister(RedirectHandlerRedirectCountMetric)

	prometheus.MustRegister(BackendHandlerRequestCountMetric)
	prometheus.MustRegister(BackendHandlerResponseDurationSecondsMetric)
}
