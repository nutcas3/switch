package network

import (
	"net"
	"sync"

	"gopherswitch/internal/modules/auth"
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
