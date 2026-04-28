package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

// Register registers all metrics with the default Prometheus registry
func (c *Collector) Register() error {
	// Transaction metrics
	if err := prometheus.Register(c.transactionsTotal); err != nil {
		return err
	}
	if err := prometheus.Register(c.transactionDuration); err != nil {
		return err
	}
	if err := prometheus.Register(c.transactionErrors); err != nil {
		return err
	}

	// Connection metrics
	if err := prometheus.Register(c.activeConnections); err != nil {
		return err
	}
	if err := prometheus.Register(c.connectionErrors); err != nil {
		return err
	}

	// Routing metrics
	if err := prometheus.Register(c.routingAttempts); err != nil {
		return err
	}
	if err := prometheus.Register(c.routingFailures); err != nil {
		return err
	}
	if err := prometheus.Register(c.routingLatency); err != nil {
		return err
	}

	// HSM metrics
	if err := prometheus.Register(c.hsmOperations); err != nil {
		return err
	}
	if err := prometheus.Register(c.hsmErrors); err != nil {
		return err
	}
	if err := prometheus.Register(c.hsmLatency); err != nil {
		return err
	}

	// Database metrics
	if err := prometheus.Register(c.dbConnections); err != nil {
		return err
	}
	if err := prometheus.Register(c.dbQueries); err != nil {
		return err
	}
	if err := prometheus.Register(c.dbQueryDuration); err != nil {
		return err
	}
	if err := prometheus.Register(c.dbErrors); err != nil {
		return err
	}

	// Settlement metrics
	if err := prometheus.Register(c.settlementBatches); err != nil {
		return err
	}
	if err := prometheus.Register(c.settlementTransactions); err != nil {
		return err
	}
	if err := prometheus.Register(c.settlementAmount); err != nil {
		return err
	}
	if err := prometheus.Register(c.settlementErrors); err != nil {
		return err
	}

	// Privacy metrics
	if err := prometheus.Register(c.privacyOperations); err != nil {
		return err
	}
	if err := prometheus.Register(c.auditProofs); err != nil {
		return err
	}
	if err := prometheus.Register(c.dataMasking); err != nil {
		return err
	}

	return nil
}

// Transaction metrics methods

func (c *Collector) IncrementTransactionCounter(mti, responseCode string) {
	c.transactionsTotal.WithLabelValues(mti, responseCode, "unknown").Inc()
}

func (c *Collector) IncrementErrorCounter(errorType string) {
	c.transactionErrors.WithLabelValues(errorType, "unknown").Inc()
}

func (c *Collector) ObserveTransactionDuration(mti string, duration float64) {
	c.transactionDuration.WithLabelValues(mti, "unknown").Observe(duration)
}

func (c *Collector) IncrementTransactionError(errorType, mti string) {
	c.transactionErrors.WithLabelValues(errorType, mti).Inc()
}

// Connection metrics methods

func (c *Collector) SetActiveConnections(count int) {
	c.activeConnections.Set(float64(count))
}

func (c *Collector) IncrementConnectionErrors() {
	c.connectionErrors.Inc()
}

// Routing metrics methods

func (c *Collector) IncrementRoutingAttempts(binPrefix, cardType string) {
	c.routingAttempts.WithLabelValues(binPrefix, cardType).Inc()
}

func (c *Collector) IncrementRoutingFailures(binPrefix, errorType string) {
	c.routingFailures.WithLabelValues(binPrefix, errorType).Inc()
}

func (c *Collector) ObserveRoutingLatency(destination string, duration time.Duration) {
	c.routingLatency.WithLabelValues(destination).Observe(duration.Seconds())
}

// HSM metrics methods

func (c *Collector) IncrementHSMOperation(operation, result string) {
	c.hsmOperations.WithLabelValues(operation, result).Inc()
}

func (c *Collector) IncrementHSMError(operation, errorType string) {
	c.hsmErrors.WithLabelValues(operation, errorType).Inc()
}

func (c *Collector) ObserveHSMLatency(operation string, duration time.Duration) {
	c.hsmLatency.WithLabelValues(operation).Observe(duration.Seconds())
}

// Database metrics methods

func (c *Collector) SetDBConnections(count int) {
	c.dbConnections.Set(float64(count))
}

func (c *Collector) IncrementDBQuery(operation, table string) {
	c.dbQueries.WithLabelValues(operation, table).Inc()
}

func (c *Collector) ObserveDBQueryDuration(operation, table string, duration time.Duration) {
	c.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

func (c *Collector) IncrementDBError(operation, errorType string) {
	c.dbErrors.WithLabelValues(operation, errorType).Inc()
}

// Settlement metrics methods

func (c *Collector) IncrementSettlementBatches(status string) {
	c.settlementBatches.WithLabelValues(status).Inc()
}

func (c *Collector) IncrementSettlementTransactions(batchDate string) {
	c.settlementTransactions.WithLabelValues(batchDate).Inc()
}

func (c *Collector) AddSettlementAmount(currency, batchDate string, amount float64) {
	c.settlementAmount.WithLabelValues(currency, batchDate).Add(amount)
}

func (c *Collector) IncrementSettlementError(errorType string) {
	c.settlementErrors.WithLabelValues(errorType).Inc()
}

// Privacy metrics methods

func (c *Collector) IncrementPrivacyOperation(operation, result string) {
	c.privacyOperations.WithLabelValues(operation, result).Inc()
}

func (c *Collector) IncrementAuditProofs(operation, result string) {
	c.auditProofs.WithLabelValues(operation, result).Inc()
}

func (c *Collector) IncrementDataMasking(dataType string) {
	c.dataMasking.WithLabelValues(dataType).Inc()
}

// StartMetricsServer starts a Prometheus metrics server
func (c *Collector) StartMetricsServer(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}
