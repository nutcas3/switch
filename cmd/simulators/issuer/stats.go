package main

import (
	"time"
)

// statsReporter periodically reports statistics
func (s *IssuerSimulator) statsReporter() {
	defer s.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
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
func (s *IssuerSimulator) reportStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	elapsed := time.Since(s.stats.startTime)
	rate := float64(s.stats.requests) / elapsed.Seconds()

	// Calculate average response time
	var avgResponseTime time.Duration
	if len(s.stats.responseTimes) > 0 {
		var total time.Duration
		for _, rt := range s.stats.responseTimes {
			total += rt
		}
		avgResponseTime = total / time.Duration(len(s.stats.responseTimes))
	}

	approvalRate := float64(0)
	if s.stats.requests > 0 {
		approvalRate = float64(s.stats.approvals) / float64(s.stats.requests) * 100
	}

	s.logger.Printf("Stats: Requests=%d, Approved=%d, Declined=%d, Errors=%d, Rate=%.2f/s, Approval=%.1f%%, AvgRT=%v",
		s.stats.requests, s.stats.approvals, s.stats.declines, s.stats.errors, rate, approvalRate, avgResponseTime)
}

// printFinalStats prints final simulation statistics
func (s *IssuerSimulator) printFinalStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	elapsed := time.Since(s.stats.startTime)
	rate := float64(s.stats.requests) / elapsed.Seconds()

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

	approvalRate := float64(0)
	if s.stats.requests > 0 {
		approvalRate = float64(s.stats.approvals) / float64(s.stats.requests) * 100
	}

	s.logger.Printf("=== Final Statistics ===")
	s.logger.Printf("Duration: %v", elapsed)
	s.logger.Printf("Requests: %d", s.stats.requests)
	s.logger.Printf("Approved: %d", s.stats.approvals)
	s.logger.Printf("Declined: %d", s.stats.declines)
	s.logger.Printf("Errors: %d", s.stats.errors)
	s.logger.Printf("Request Rate: %.2f/s", rate)
	s.logger.Printf("Approval Rate: %.1f%%", approvalRate)
	s.logger.Printf("Response Times - Avg: %v, Min: %v, Max: %v", avgResponseTime, minResponseTime, maxResponseTime)
}
