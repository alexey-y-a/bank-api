package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var HTTPRequestTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_request_total",
		Help: "Total number of HTTP requests by method, path and status code",
	},
	[]string{"method", "path", "status"},
)

var HTTPRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Help:    "HTTP request duration in seconds by method and path.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "path"},
)
