package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/moov-io/iso8583"
)

// processRequest processes an incoming ISO 8583 request
func (s *IssuerSimulator) processRequest(data []byte) ([]byte, error) {
	start := time.Now()

	// Unpack message
	msg := iso8583.NewMessage(s.spec)
	if err := msg.Unpack(data); err != nil {
		return nil, fmt.Errorf("failed to unpack message: %w", err)
	}

	// Get MTI
	mti, err := msg.GetMTI()
	if err != nil {
		return nil, fmt.Errorf("failed to get MTI: %w", err)
	}

	// Get PAN
	pan, err := msg.GetString(2)
	if err != nil {
		return nil, fmt.Errorf("failed to get PAN: %w", err)
	}

	// Get amount
	amountStr, err := msg.GetString(4)
	if err != nil {
		return nil, fmt.Errorf("failed to get amount: %w", err)
	}

	// Get RRN
	rrn, _ := msg.GetString(37)

	s.logger.Printf("Processing request: MTI=%s, PAN=%s, Amount=%s, RRN=%s",
		mti, maskPAN(pan), amountStr, rrn)

	// Simulate processing delay
	delay := s.config.MinDelay
	if s.config.MaxDelay > s.config.MinDelay {
		delay += time.Duration(rand.Int63n(int64(s.config.MaxDelay - s.config.MinDelay)))
	}
	time.Sleep(delay)

	// Create response message
	response := iso8583.NewMessage(s.spec)

	// Determine response based on approval rate
	approve := rand.Float64() < s.config.ApprovalRate
	var responseCode string

	if approve {
		responseCode = "00" // Approved
		s.stats.mu.Lock()
		s.stats.approvals++
		s.stats.mu.Unlock()
	} else {
		// Random decline reason
		responseCodes := []string{"51", "55", "61", "65", "91"}
		responseCode = responseCodes[rand.Intn(len(responseCodes))]
		s.stats.mu.Lock()
		s.stats.declines++
		s.stats.mu.Unlock()
	}

	// Set response code
	response.Field(39, responseCode)

	// Add authorization ID if approved
	if approve {
		authID := fmt.Sprintf("%06d", rand.Intn(999999))
		response.Field(38, authID)
	}

	// Add balance if available
	if approve {
		balance := s.config.Balance
		balanceStr := fmt.Sprintf("01%012d", balance)
		response.Field(54, balanceStr)
	}

	// Pack response
	responseData, err := response.Pack()
	if err != nil {
		return nil, fmt.Errorf("failed to pack response: %w", err)
	}

	// Update stats
	responseTime := time.Since(start)
	s.stats.mu.Lock()
	s.stats.requests++
	s.stats.responseTimes = append(s.stats.responseTimes, responseTime)
	s.stats.mu.Unlock()

	s.logger.Printf("Response: RRN=%s, Code=%s, Time=%v", rrn, responseCode, responseTime)

	return responseData, nil
}
