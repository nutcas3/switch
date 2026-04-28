package metrics

import "time"

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
