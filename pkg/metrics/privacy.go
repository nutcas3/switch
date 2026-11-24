package metrics

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
