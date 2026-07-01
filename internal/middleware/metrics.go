package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request duration",
		},
		[]string{"method", "path"},
	)

	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total HTTP error responses",
		},
		[]string{"method", "path", "status"},
	)

	registerMetricsOnce sync.Once
)

func RegisterMetrics() {
	registerMetricsOnce.Do(func() {
		prometheus.MustRegister(RequestsTotal, RequestDuration, ErrorsTotal)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		RequestsTotal.WithLabelValues(
			r.Method,
			routePattern(r),
			strconv.Itoa(rw.status),
		).Inc()
		if rw.status >= 400 {
			ErrorsTotal.WithLabelValues(
				r.Method,
				routePattern(r),
				strconv.Itoa(rw.status),
			).Inc()
		}

		RequestDuration.WithLabelValues(
			r.Method,
			routePattern(r),
		).Observe(duration)
	})
}

func routePattern(r *http.Request) string {
	if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
		if pattern := routeContext.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return r.URL.Path
}
