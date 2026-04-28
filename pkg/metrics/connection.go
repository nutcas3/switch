package metrics

// Connection metrics methods

func (c *Collector) SetActiveConnections(count int) {
	c.activeConnections.Set(float64(count))
}

func (c *Collector) IncrementConnectionErrors() {
	c.connectionErrors.Inc()
}
