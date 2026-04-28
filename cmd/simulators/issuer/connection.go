package main

import (
	"fmt"
	"net"
	"time"
)

// handleConnection handles a single client connection
func (s *IssuerSimulator) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	s.logger.Printf("New connection from: %s", conn.RemoteAddr())

	// Set timeouts
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

	for {
		select {
		case <-s.stopChan:
			return
		default:
			// Read message length
			lengthBuf := make([]byte, 2)
			_, err := conn.Read(lengthBuf)
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
				return
			}

			// Read message
			buf := make([]byte, msgLen)
			_, err = conn.Read(buf)
			if err != nil {
				s.logger.Printf("Error reading message: %v", err)
				return
			}

			// Process request
			response, err := s.processRequest(buf)
			if err != nil {
				s.logger.Printf("Error processing request: %v", err)
				s.stats.mu.Lock()
				s.stats.errors++
				s.stats.mu.Unlock()
				continue
			}

			// Send response
			if err := s.sendResponse(conn, response); err != nil {
				s.logger.Printf("Error sending response: %v", err)
				return
			}

			// Update read deadline for keep-alive
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		}
	}
}

// sendResponse sends a response back to the client
func (s *IssuerSimulator) sendResponse(conn net.Conn, data []byte) error {
	// Add length prefix
	length := len(data)
	lengthBuf := []byte{byte(length >> 8), byte(length & 0xFF)}
	fullMessage := append(lengthBuf, data...)

	// Send response
	_, err := conn.Write(fullMessage)
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}
