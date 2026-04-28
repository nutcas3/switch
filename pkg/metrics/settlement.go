package metrics

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
