package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "kubeai_http_requests_total",
		Help: "Total number of HTTP requests handled by the KubeAI API.",
	},
	[]string{"method", "path", "status"},
)

var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "kubeai_http_request_duration_seconds",
		Help:    "Duration of HTTP requests handled by the KubeAI API.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "path"},
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         0,
		}

		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		// Use the registered route pattern, not the raw URL path,
		// to avoid creating a metric series for every run name.
		path := r.Pattern
		if path == "" {
			path = "unmatched"
		}

		method := r.Method
		statusCode := strconv.Itoa(status)

		httpRequestsTotal.WithLabelValues(
			method, path, statusCode,
		).Inc()

		httpRequestDuration.WithLabelValues(
			method, path,
		).Observe(time.Since(start).Seconds())
	})
}
