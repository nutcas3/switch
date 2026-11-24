package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Collector implements metrics collection using Prometheus
type Collector struct {
	// Transaction metrics
	transactionsTotal   *prometheus.CounterVec
	transactionDuration *prometheus.HistogramVec
	transactionErrors   *prometheus.CounterVec

	// Connection metrics
	activeConnections prometheus.Gauge
	connectionErrors  prometheus.Counter

	// Routing metrics
	routingAttempts *prometheus.CounterVec
	routingFailures *prometheus.CounterVec
	routingLatency  *prometheus.HistogramVec

	// HSM metrics
	hsmOperations *prometheus.CounterVec
	hsmErrors     *prometheus.CounterVec
	hsmLatency    *prometheus.HistogramVec

	// Database metrics
	dbConnections   prometheus.Gauge
	dbQueries       *prometheus.CounterVec
	dbQueryDuration *prometheus.HistogramVec
	dbErrors        *prometheus.CounterVec

	// Settlement metrics
	settlementBatches      *prometheus.CounterVec
	settlementTransactions *prometheus.CounterVec
	settlementAmount       *prometheus.CounterVec
	settlementErrors       *prometheus.CounterVec

	// Privacy metrics
	privacyOperations *prometheus.CounterVec
	auditProofs       *prometheus.CounterVec
	dataMasking       *prometheus.CounterVec
}
