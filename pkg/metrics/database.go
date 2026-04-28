package metrics

import "time"

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
