package httptransport

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce sync.Once

	httpRequestsTotal *prometheus.CounterVec
	httpErrorsTotal   *prometheus.CounterVec
	httpLatency       *prometheus.HistogramVec
)

func metricsMiddleware(next http.Handler) http.Handler {
	initMetrics()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		method := r.Method
		route := normalizeRoute(r.URL.Path)
		status := strconv.Itoa(recorder.statusCode)

		httpRequestsTotal.WithLabelValues(method, route, status).Inc()
		httpLatency.WithLabelValues(method, route, status).Observe(time.Since(startedAt).Seconds())
		if recorder.statusCode >= 400 {
			httpErrorsTotal.WithLabelValues(method, route, status).Inc()
		}
	})
}

func initMetrics() {
	metricsOnce.Do(func() {
		httpRequestsTotal = registerCounterVec(prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "app_http_requests_total",
				Help: "Total number of HTTP requests handled by the service.",
			},
			[]string{"method", "route", "status"},
		))

		httpErrorsTotal = registerCounterVec(prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "app_http_errors_total",
				Help: "Total number of HTTP requests that finished with status >= 400.",
			},
			[]string{"method", "route", "status"},
		))

		httpLatency = registerHistogramVec(prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "app_http_request_duration_seconds",
				Help:    "HTTP request latency in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		))
	})
}

func registerCounterVec(counter *prometheus.CounterVec) *prometheus.CounterVec {
	if err := prometheus.Register(counter); err != nil {
		if alreadyRegisteredErr, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, castOK := alreadyRegisteredErr.ExistingCollector.(*prometheus.CounterVec); castOK {
				return existing
			}
		}
		panic(err)
	}
	return counter
}

func registerHistogramVec(histogram *prometheus.HistogramVec) *prometheus.HistogramVec {
	if err := prometheus.Register(histogram); err != nil {
		if alreadyRegisteredErr, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, castOK := alreadyRegisteredErr.ExistingCollector.(*prometheus.HistogramVec); castOK {
				return existing
			}
		}
		panic(err)
	}
	return histogram
}

func normalizeRoute(path string) string {
	switch {
	case strings.HasPrefix(path, "/v1/catalog/"):
		return "/v1/catalog/*"
	case strings.HasPrefix(path, "/v1/booking/"):
		return "/v1/booking/*"
	case strings.HasPrefix(path, "/v1/ticketing/"):
		return "/v1/ticketing/*"
	case strings.HasPrefix(path, "/swagger/specs/"):
		return "/swagger/specs/*"
	case strings.HasPrefix(path, "/debug/pprof/"):
		return "/debug/pprof/*"
	default:
		return path
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
