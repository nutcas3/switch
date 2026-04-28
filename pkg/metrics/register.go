package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

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
