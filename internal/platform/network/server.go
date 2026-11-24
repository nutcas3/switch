package network

import (
	"context"
	"fmt"
	"net"
)

// Start starts the TCP server on the specified port
func (s *TCPServer) Start(port string) error {
	var err error
	s.listener, err = net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}

	s.logger.Info("TCP server started", map[string]interface{}{
		"port": port,
	})

	// Start connection acceptor goroutine
	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// acceptConnections accepts new connections
func (s *TCPServer) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.shutdownChan:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.shutdownChan:
					return
				default:
					s.logger.Error("Failed to accept connection", err, nil)
					continue
				}
			}

			// Handle connection in a new goroutine
			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

// Shutdown gracefully shuts down the server
func (s *TCPServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down TCP server", nil)

	// Signal shutdown
	close(s.shutdownChan)

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for all connections to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("TCP server shutdown complete", nil)
		return nil
	case <-ctx.Done():
		s.logger.Warn("TCP server shutdown timeout", nil)
		return fmt.Errorf("shutdown timeout")
	}
}

// GetStats returns server statistics
func (s *TCPServer) GetStats() map[string]interface{} {
	return map[string]any{
		"active_connections": s.connPool.Size(),
		"max_connections":    s.connPool.maxSize,
	}
}
