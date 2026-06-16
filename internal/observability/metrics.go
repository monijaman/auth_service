package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "auth",
		Name:      "http_requests_total",
		Help:      "Total HTTP requests by method, path, and status.",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "auth",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})

	GRPCRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "auth",
		Name:      "grpc_requests_total",
		Help:      "Total gRPC requests by method and status.",
	}, []string{"method", "status"})

	ActiveUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "auth",
		Name:      "active_users_total",
		Help:      "Total number of active (non-deleted) users.",
	})

	TokensIssued = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "auth",
		Name:      "tokens_issued_total",
		Help:      "Total JWT tokens issued by type.",
	}, []string{"type"})

	LoginAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "auth",
		Name:      "login_attempts_total",
		Help:      "Total login attempts by result.",
	}, []string{"result"}) // success | failure
)
