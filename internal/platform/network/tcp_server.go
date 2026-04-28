package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"gopherswitch/internal/modules/auth"

	"github.com/moov-io/iso8583"
)

// TCPServer handles incoming TCP connections for ISO 8583 messages
type TCPServer struct {
	listener     net.Listener
	authService  *auth.Service
	logger       Logger
	metrics      MetricsCollector
	connPool     *ConnectionPool
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

// Logger interface for structured logging
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Debug(msg string, fields map[string]interface{})
}

// MetricsCollector interface for metrics
type MetricsCollector interface {
	IncrementTransactionCounter(mti string, responseCode string)
	ObserveTransactionDuration(mti string, duration float64)
	SetActiveConnections(count int)
	IncrementErrorCounter(errorType string)
}

// ConnectionPool manages active connections
type ConnectionPool struct {
	connections map[string]*net.Conn
	mutex       sync.RWMutex
	maxSize     int
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int) *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]*net.Conn),
		maxSize:     maxSize,
	}
}

// Add adds a connection to the pool
func (cp *ConnectionPool) Add(id string, conn *net.Conn) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if len(cp.connections) >= cp.maxSize {
		return fmt.Errorf("connection pool is full")
	}

	cp.connections[id] = conn
	return nil
}

// Remove removes a connection from the pool
func (cp *ConnectionPool) Remove(id string) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	delete(cp.connections, id)
}

// Size returns the current pool size
func (cp *ConnectionPool) Size() int {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return len(cp.connections)
}

// NewTCPServer creates a new TCP server
func NewTCPServer(authService *auth.Service, logger Logger, metrics MetricsCollector) *TCPServer {
	return &TCPServer{
		authService:  authService,
		logger:       logger,
		metrics:      metrics,
		connPool:     NewConnectionPool(1000), // Max 1000 concurrent connections
		shutdownChan: make(chan struct{}),
	}
}

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

// handleConnection handles a single client connection
func (s *TCPServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	connID := conn.RemoteAddr().String()

	s.logger.Info("New connection established", map[string]interface{}{
		"remote_addr": connID,
	})

	// Add to connection pool
	if err := s.connPool.Add(connID, &conn); err != nil {
		s.logger.Warn("Failed to add connection to pool", map[string]interface{}{
			"remote_addr": connID,
			"error":       err.Error(),
		})
		conn.Close()
		return
	}

	defer s.connPool.Remove(connID)

	// Update metrics
	s.metrics.SetActiveConnections(s.connPool.Size())

	// Set connection timeouts
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

	// Handle messages
	for {
		select {
		case <-s.shutdownChan:
			return
		default:
			// Read message length (2 bytes for ISO 8583)
			lengthBuf := make([]byte, 2)
			_, err := io.ReadFull(conn, lengthBuf)
			if err != nil {
				if err != io.EOF {
					s.logger.Error("Failed to read message length", err, map[string]interface{}{
						"remote_addr": connID,
					})
				}
				return
			}

			// Parse message length (big endian)
			msgLen := int(lengthBuf[0])<<8 | int(lengthBuf[1])
			if msgLen <= 0 || msgLen > 8192 { // Max 8KB message
				s.logger.Warn("Invalid message length", map[string]interface{}{
					"remote_addr": connID,
					"message_len": msgLen,
				})
				s.metrics.IncrementErrorCounter("invalid_message_length")
				return
			}

			// Read message
			msgBuf := make([]byte, msgLen)
			_, err = io.ReadFull(conn, msgBuf)
			if err != nil {
				s.logger.Error("Failed to read message", err, map[string]interface{}{
					"remote_addr": connID,
				})
				return
			}

			// Process message
			response, err := s.processMessage(msgBuf)
			if err != nil {
				s.logger.Error("Failed to process message", err, map[string]interface{}{
					"remote_addr": connID,
				})
				s.metrics.IncrementErrorCounter("message_processing_error")
				continue
			}

			// Send response
			if err := s.sendResponse(conn, response); err != nil {
				s.logger.Error("Failed to send response", err, map[string]interface{}{
					"remote_addr": connID,
				})
				return
			}

			// Update read deadline for keep-alive
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		}
	}
}

// processMessage processes an ISO 8583 message
func (s *TCPServer) processMessage(data []byte) ([]byte, error) {
	startTime := time.Now()

	// Parse ISO 8583 message
	spec := iso8583.Spec87
	message := iso8583.NewMessage(spec)

	if err := message.Unpack(data); err != nil {
		return nil, fmt.Errorf("failed to unpack ISO 8583 message: %w", err)
	}

	// Get MTI
	mti, err := message.GetMTI()
	if err != nil {
		return nil, fmt.Errorf("failed to get MTI: %w", err)
	}

	s.logger.Debug("Processing message", map[string]interface{}{
		"mti": mti,
	})

	// Process based on MTI
	ctx := context.Background()
	var response *iso8583.Message

	switch mti {
	case "0100": // Authorization request
		response, err = s.authService.ProcessAuthorization(ctx, message)
	case "0400": // Reversal request
		response, err = s.authService.ProcessReversal(ctx, message)
	default:
		return nil, fmt.Errorf("unsupported MTI: %s", mti)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to process %s message: %w", mti, err)
	}

	// Pack response
	responseData, err := response.Pack()
	if err != nil {
		return nil, fmt.Errorf("failed to pack response: %w", err)
	}

	// Record metrics
	duration := time.Since(startTime).Seconds()
	s.metrics.ObserveTransactionDuration(mti, duration)

	return responseData, nil
}

// sendResponse sends a response back to the client
func (s *TCPServer) sendResponse(conn net.Conn, data []byte) error {
	// Add length prefix (2 bytes, big endian)
	length := len(data)
	lengthBuf := []byte{byte(length >> 8), byte(length & 0xFF)}

	// Send length + data
	if _, err := conn.Write(append(lengthBuf, data...)); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
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
	return map[string]interface{}{
		"active_connections": s.connPool.Size(),
		"max_connections":    s.connPool.maxSize,
	}
}
