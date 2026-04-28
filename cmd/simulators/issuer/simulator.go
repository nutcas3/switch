package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/moov-io/iso8583"
)

// Config holds issuer simulator configuration
type Config struct {
	Port         int
	ApprovalRate float64 // 0.0 to 1.0
	MinDelay     time.Duration
	MaxDelay     time.Duration
	Balance      int64
}

// IssuerSimulator simulates an issuer bank
type IssuerSimulator struct {
	config   Config
	spec     *iso8583.MessageSpec
	logger   *log.Logger
	stats    *Stats
	listener net.Listener
	wg       sync.WaitGroup
	stopChan chan struct{}
}

// Stats tracks simulator statistics
type Stats struct {
	mu              sync.Mutex
	requests        int64
	approvals       int64
	declines        int64
	errors          int64
	startTime       time.Time
	responseTimes   []time.Duration
	avgResponseTime time.Duration
}

// NewIssuerSimulator creates a new issuer simulator
func NewIssuerSimulator(config Config) *IssuerSimulator {
	return &IssuerSimulator{
		config:   config,
		spec:     iso8583.Spec87,
		logger:   log.New(os.Stdout, "[ISSUER] ", log.LstdFlags),
		stats:    &Stats{startTime: time.Now()},
		stopChan: make(chan struct{}),
	}
}

// Run starts the issuer simulator
func (s *IssuerSimulator) Run() error {
	s.logger.Printf("Starting issuer simulator")
	s.logger.Printf("Port: %d", s.config.Port)
	s.logger.Printf("Approval Rate: %.1f%%", s.config.ApprovalRate*100)
	s.logger.Printf("Response Delay: %v - %v", s.config.MinDelay, s.config.MaxDelay)

	// Start TCP listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	defer listener.Close()

	s.listener = listener
	s.logger.Printf("Issuer simulator listening on port %d", s.config.Port)

	// Start stats reporter
	s.wg.Add(1)
	go s.statsReporter()

	// Accept connections
	for {
		select {
		case <-s.stopChan:
			return nil
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.stopChan:
					return nil
				default:
					s.logger.Printf("Error accepting connection: %v", err)
					continue
				}
			}

			// Handle connection in a new goroutine
			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

// Shutdown gracefully shuts down the simulator
func (s *IssuerSimulator) Shutdown() {
	s.logger.Println("Shutting down issuer simulator...")
	close(s.stopChan)

	if s.listener != nil {
		s.listener.Close()
	}

	s.wg.Wait()

	// Print final stats
	s.printFinalStats()
}
