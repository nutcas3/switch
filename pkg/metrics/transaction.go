package metrics

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
