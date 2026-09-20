// Package metrics provides Prometheus metrics for the application.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal counts total HTTP requests by method, path, and status.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration tracks request latency in seconds.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets, // [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
		},
		[]string{"method", "path"},
	)

	// HTTPRequestsInFlight tracks current in-flight requests.
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)

	// DBConnectionsActive tracks active DB connections.
	DBConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)

	// DBConnectionsIdle tracks idle DB connections.
	DBConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		},
	)
)

// PostOperation is the bounded set of values for the blog_post_writes_total "operation" label.
type PostOperation string

const (
	PostOpCreate    PostOperation = "create"
	PostOpUpdate    PostOperation = "update"
	PostOpDelete    PostOperation = "delete"
	PostOpPublish   PostOperation = "publish"
	PostOpUnpublish PostOperation = "unpublish"
)

var (
	// PostWritesTotal counts successful post writes by operation.
	// Never label by slug, post id or user: those are unbounded.
	PostWritesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "blog_post_writes_total",
			Help: "Total number of successful blog post writes by operation",
		},
		[]string{"operation"},
	)

	// PostSlugAttempts records how many slug candidates a create tried before one was free.
	// Values above 1 mean slug collisions; the service gives up at 100.
	PostSlugAttempts = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "blog_post_slug_attempts",
			Help:    "Slug candidates tried per successful post creation",
			Buckets: []float64{1, 2, 3, 5, 10, 25, 50, 100},
		},
	)
)
