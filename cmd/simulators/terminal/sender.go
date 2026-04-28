package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/moov-io/iso8583"
)

// messageSender sends messages at the configured rate
func (s *Simulator) messageSender() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Second / time.Duration(s.config.TPS))
	defer ticker.Stop()

	sent := 0
	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			if s.config.Count > 0 && sent >= s.config.Count {
				s.stopChan <- struct{}{}
				return
			}

			if err := s.sendMessage(); err != nil {
				s.logger.Printf("Error sending message: %v", err)
				s.stats.mu.Lock()
				s.stats.errors++
				s.stats.mu.Unlock()
			} else {
				sent++
				s.stats.mu.Lock()
				s.stats.sent++
				s.stats.lastSentTime = time.Now()
				s.stats.mu.Unlock()
			}
		}
	}
}

// sendMessage creates and sends an ISO 8583 message
func (s *Simulator) sendMessage() error {
	// Create message
	msg := iso8583.NewMessage(s.spec)

	// Set MTI (0100 - Authorization Request)
	msg.MTI("0100")

	// Set PAN (random test card)
	pan := s.config.TestPANs[rand.Intn(len(s.config.TestPANs))]
	msg.Field(2, pan)

	// Set Processing Code (00 - Purchase)
	msg.Field(3, "00")

	// Set Amount (random test amount)
	amount := s.config.TestAmounts[rand.Intn(len(s.config.TestAmounts))]
	msg.Field(4, fmt.Sprintf("%012d", amount))

	// Set Transmission DateTime
	msg.Field(7, time.Now().Format("010215304"))

	// Set STAN (System Trace Audit Number)
	stan := fmt.Sprintf("%06d", rand.Intn(999999))
	msg.Field(11, stan)

	// Set RRN (Retrieval Reference Number)
	rrn := fmt.Sprintf("%012d", rand.Intn(999999999999))
	msg.Field(37, rrn)

	// Set Terminal ID
	msg.Field(41, "12345678")

	// Set Merchant ID
	msg.Field(42, "TESTMERCHANT001")

	// Set Acquiring Institution ID
	msg.Field(32, "123456")

	// Set Currency Code
	msg.Field(49, "840") // USD

	// Set Merchant Type
	msg.Field(18, "5999") // Other Services

	// Pack message
	data, err := msg.Pack()
	if err != nil {
		return fmt.Errorf("failed to pack message: %w", err)
	}

	// Add length prefix
	length := len(data)
	lengthBuf := []byte{byte(length >> 8), byte(length & 0xFF)}
	fullMessage := append(lengthBuf, data...)

	// Send message
	_, err = s.conn.Write(fullMessage)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
