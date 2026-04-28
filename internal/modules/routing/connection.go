package routing

import (
	"context"
	"fmt"
	"net"
	"time"

	"gopherswitch/internal/modules/auth/domain"
)

// SendToIssuer forwards a message to the issuer
func (s *Service) SendToIssuer(_ context.Context, message []byte, destination *domain.RoutingRule) ([]byte, error) {
	// Create connection key
	connKey := fmt.Sprintf("%s:%d", destination.DestinationHost, destination.DestinationPort)

	// Get or create connection
	conn, err := s.getConnection(connKey, destination.DestinationHost, destination.DestinationPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Set write deadline
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// Send message (add length prefix)
	length := len(message)
	lengthBuf := []byte{byte(length >> 8), byte(length & 0xFF)}
	fullMessage := append(lengthBuf, message...)

	_, err = conn.Write(fullMessage)
	if err != nil {
		s.connMutex.Lock()
		delete(s.connections, connKey)
		s.connMutex.Unlock()
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Read response length
	responseLengthBuf := make([]byte, 2)
	_, err = conn.Read(responseLengthBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read response length: %w", err)
	}

	responseLength := int(responseLengthBuf[0])<<8 | int(responseLengthBuf[1])
	if responseLength <= 0 || responseLength > 8192 {
		return nil, fmt.Errorf("invalid response length: %d", responseLength)
	}

	// Read response
	response := make([]byte, responseLength)
	_, err = conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	s.logger.Debug("Message routed successfully", map[string]interface{}{
		"destination":  connKey,
		"request_len":  len(message),
		"response_len": len(response),
	})

	return response, nil
}

// getConnection gets or creates a connection to the destination
func (s *Service) getConnection(key, host string, port int) (net.Conn, error) {
	s.connMutex.RLock()
	conn, exists := s.connections[key]
	s.connMutex.RUnlock()

	if exists && !isConnectionClosed(conn) {
		return conn, nil
	}

	// Create new connection
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	// Double-check after acquiring write lock
	if conn, exists := s.connections[key]; exists && !isConnectionClosed(conn) {
		return conn, nil
	}

	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	newConn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	s.connections[key] = newConn

	s.logger.Info("Created new routing connection", map[string]interface{}{
		"address":           address,
		"total_connections": len(s.connections),
	})

	return newConn, nil
}

// isConnectionClosed checks if a connection is closed
func isConnectionClosed(conn net.Conn) bool {
	// Try to set a zero-length deadline to check if connection is alive
	conn.SetReadDeadline(time.Now())
	var buf [1]byte
	_, err := conn.Read(buf[:])
	if err != nil {
		return true
	}
	return false
}

// Close closes all connections
func (s *Service) Close() error {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	var errors []error
	for key, conn := range s.connections {
		if err := conn.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close connection %s: %w", key, err))
		}
		delete(s.connections, key)
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing connections: %v", errors)
	}

	return nil
}
