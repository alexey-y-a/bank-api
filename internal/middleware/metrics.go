package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/alexey-y-a/bank-api/pkg/metrics"
)

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		mw := newMetricsResponseWriter(w)
		next.ServeHTTP(mw, r)
		duration := time.Since(start)
		status := strconv.Itoa(mw.statusCode)
		metrics.HTTPRequestTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration.Seconds())
	})
}
