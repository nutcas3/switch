package main

import (
	"net"
	"time"

	"github.com/moov-io/iso8583"
)

// responseReceiver receives and processes responses
func (s *Simulator) responseReceiver() {
	defer s.wg.Done()

	buf := make([]byte, 8192)
	for {
		select {
		case <-s.stopChan:
			return
		default:
			// Read message length
			lengthBuf := make([]byte, 2)
			_, err := s.conn.Read(lengthBuf)
			if err != nil {
				if err != net.ErrClosed {
					s.logger.Printf("Error reading length: %v", err)
				}
				return
			}

			// Parse message length
			msgLen := int(lengthBuf[0])<<8 | int(lengthBuf[1])
			if msgLen <= 0 || msgLen > 8192 {
				s.logger.Printf("Invalid message length: %d", msgLen)
				continue
			}

			// Read message
			_, err = s.conn.Read(buf[:msgLen])
			if err != nil {
				s.logger.Printf("Error reading message: %v", err)
				return
			}

			// Process response
			s.processResponse(buf[:msgLen])
		}
	}
}

// processResponse processes an incoming response
func (s *Simulator) processResponse(data []byte) {
	start := time.Now()

	// Unpack message
	msg := iso8583.NewMessage(s.spec)
	if err := msg.Unpack(data); err != nil {
		s.logger.Printf("Error unpacking response: %v", err)
		s.stats.mu.Lock()
		s.stats.errors++
		s.stats.mu.Unlock()
		return
	}

	// Get response code
	responseCode, err := msg.GetString(39)
	if err != nil {
		s.logger.Printf("Error getting response code: %v", err)
		return
	}

	// Get RRN
	rrn, _ := msg.GetString(37)

	// Calculate response time
	responseTime := time.Since(start)

	// Update stats
	s.stats.mu.Lock()
	s.stats.received++
	s.stats.responseTimes = append(s.stats.responseTimes, responseTime)
	s.stats.mu.Unlock()

	// Log response
	s.logger.Printf("Response: RRN=%s, Code=%s, Time=%v", rrn, responseCode, responseTime)
}
