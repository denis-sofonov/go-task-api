package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func InstrumentHTTP(registerer prometheus.Registerer, next http.Handler) http.Handler {
	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of completed HTTP requests by route.",
		},
		[]string{"route"},
	)
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request handling duration in seconds by route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route"})
	registerer.MustRegister(requests, duration)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		elapsed := time.Since(start).Seconds()

		route := r.Pattern
		if route == "GET /metrics" {
			return
		}
		if route == "" {
			route = "unmatched"
		}

		requests.WithLabelValues(route).Inc()
		duration.WithLabelValues(route).Observe(elapsed)
	})
}
