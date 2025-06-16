package monitoring

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTPRequestsTotal is a counter for total HTTP requests.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agbaravoip_http_requests_total",
			Help: "Total number of HTTP requests made.",
		},
		[]string{"method", "path", "status_code"}, // Labels
	)

	// HTTPRequestDuration is a histogram for HTTP request latencies.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "agbaravoip_http_request_duration_seconds",
			Help:    "Histogram of HTTP request latencies.",
			Buckets: prometheus.DefBuckets, // Default buckets
		},
		[]string{"method", "path"},
	)

	// Add more metrics as needed:
	// - Active WebSocket connections (Gauge) if using WebSockets
	// - ESL command counters/durations (Counter/Histogram)
	// - Database query durations (Histogram)
)

// PrometheusMiddleware returns a Gin middleware that records Prometheus metrics for HTTP requests.
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Serve the request
		c.Next() // Important: call Next() before recording metrics that depend on the response

		// After request has been processed by other handlers
		path := c.FullPath() // Get the matched route path (e.g., /api/v1/accounts/:account_sid/calls)
		if path == "" {
			// If FullPath is empty, it might be a 404 or a route not handled by Gin's router.
			// Using c.Request.URL.Path might provide more detail for unhandled routes.
			path = c.Request.URL.Path
			if len(path) > 50 { // Avoid overly long paths for cardinality reasons
				path = path[:50]
			}
		}
		method := c.Request.Method
		statusCode := c.Writer.Status() // Get status code after c.Next()

		// Record request duration
		duration := time.Since(startTime).Seconds()
		HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)

		// Record total requests
		HTTPRequestsTotal.WithLabelValues(method, path, strconv.Itoa(statusCode)).Inc()
	}
}

// PrometheusHandler returns a Gin handler function for exposing Prometheus metrics.
func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// InitMetrics (Optional) - if you need to register metrics that are not auto-registered by promauto.
// For now, promauto handles registration of the defined metrics.
// func InitMetrics() {
//   prometheus.MustRegister(HTTPRequestsTotal)
//   prometheus.MustRegister(HTTPRequestDuration)
// }
