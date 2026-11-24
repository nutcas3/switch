package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// NewCollector creates a new Prometheus metrics collector
func NewCollector() *Collector {
	return &Collector{
		// Transaction metrics
		transactionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_transactions_total",
				Help: "Total number of transactions processed",
			},
			[]string{"mti", "response_code", "card_type"},
		),
		transactionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gopherswitch_transaction_duration_seconds",
				Help:    "Transaction processing duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"mti", "response_code"},
		),
		transactionErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_transaction_errors_total",
				Help: "Total number of transaction processing errors",
			},
			[]string{"error_type", "mti"},
		),

		// Connection metrics
		activeConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gopherswitch_active_connections",
				Help: "Number of active TCP connections",
			},
		),
		connectionErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "gopherswitch_connection_errors_total",
				Help: "Total number of connection errors",
			},
		),

		// Routing metrics
		routingAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_routing_attempts_total",
				Help: "Total number of routing attempts",
			},
			[]string{"bin_prefix", "card_type"},
		),
		routingFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_routing_failures_total",
				Help: "Total number of routing failures",
			},
			[]string{"bin_prefix", "error_type"},
		),
		routingLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gopherswitch_routing_duration_seconds",
				Help:    "Message routing duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"destination"},
		),

		// HSM metrics
		hsmOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_hsm_operations_total",
				Help: "Total number of HSM operations",
			},
			[]string{"operation", "result"},
		),
		hsmErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_hsm_errors_total",
				Help: "Total number of HSM errors",
			},
			[]string{"operation", "error_type"},
		),
		hsmLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gopherswitch_hsm_duration_seconds",
				Help:    "HSM operation duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),

		// Database metrics
		dbConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gopherswitch_db_connections",
				Help: "Number of active database connections",
			},
		),
		dbQueries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "table"},
		),
		dbQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gopherswitch_db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "table"},
		),
		dbErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_db_errors_total",
				Help: "Total number of database errors",
			},
			[]string{"operation", "error_type"},
		),

		// Settlement metrics
		settlementBatches: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_settlement_batches_total",
				Help: "Total number of settlement batches processed",
			},
			[]string{"status"},
		),
		settlementTransactions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_settlement_transactions_total",
				Help: "Total number of transactions in settlement batches",
			},
			[]string{"batch_date"},
		),
		settlementAmount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_settlement_amount_total",
				Help: "Total amount settled in cents",
			},
			[]string{"currency", "batch_date"},
		),
		settlementErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_settlement_errors_total",
				Help: "Total number of settlement errors",
			},
			[]string{"error_type"},
		),

		// Privacy metrics
		privacyOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_privacy_operations_total",
				Help: "Total number of privacy operations",
			},
			[]string{"operation", "result"},
		),
		auditProofs: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_audit_proofs_total",
				Help: "Total number of audit proofs created/verified",
			},
			[]string{"operation", "result"},
		),
		dataMasking: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gopherswitch_data_masking_total",
				Help: "Total number of data masking operations",
			},
			[]string{"data_type"},
		),
	}
}
