package metrics

import "time"

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
