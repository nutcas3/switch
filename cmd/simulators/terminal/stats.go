package main

import "time"

// statsReporter periodically reports statistics
func (s *Simulator) statsReporter() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.reportStats()
		}
	}
}

// reportStats reports current statistics
func (s *Simulator) reportStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	elapsed := time.Since(s.stats.startTime)
	currentTPS := float64(s.stats.sent) / elapsed.Seconds()

	// Calculate average response time
	var avgResponseTime time.Duration
	if len(s.stats.responseTimes) > 0 {
		var total time.Duration
		for _, rt := range s.stats.responseTimes {
			total += rt
		}
		avgResponseTime = total / time.Duration(len(s.stats.responseTimes))
	}

	s.logger.Printf("Stats: Sent=%d, Received=%d, Errors=%d, TPS=%.2f, AvgRT=%v",
		s.stats.sent, s.stats.received, s.stats.errors, currentTPS, avgResponseTime)
}

// printFinalStats prints final simulation statistics
func (s *Simulator) printFinalStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	elapsed := time.Since(s.stats.startTime)
	avgTPS := float64(s.stats.sent) / elapsed.Seconds()

	// Calculate response time statistics
	var avgResponseTime, minResponseTime, maxResponseTime time.Duration
	if len(s.stats.responseTimes) > 0 {
		var total time.Duration
		minResponseTime = s.stats.responseTimes[0]
		maxResponseTime = s.stats.responseTimes[0]

		for _, rt := range s.stats.responseTimes {
			total += rt
			if rt < minResponseTime {
				minResponseTime = rt
			}
			if rt > maxResponseTime {
				maxResponseTime = rt
			}
		}
		avgResponseTime = total / time.Duration(len(s.stats.responseTimes))
	}

	s.logger.Printf("=== Final Statistics ===")
	s.logger.Printf("Duration: %v", elapsed)
	s.logger.Printf("Sent: %d", s.stats.sent)
	s.logger.Printf("Received: %d", s.stats.received)
	s.logger.Printf("Errors: %d", s.stats.errors)
	s.logger.Printf("Average TPS: %.2f", avgTPS)
	s.logger.Printf("Success Rate: %.2f%%", float64(s.stats.received)/float64(s.stats.sent)*100)
	s.logger.Printf("Response Times - Avg: %v, Min: %v, Max: %v", avgResponseTime, minResponseTime, maxResponseTime)
}
