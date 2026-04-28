package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/moov-io/iso8583"
)

// Simulator generates and sends ISO 8583 messages
type Simulator struct {
	config   Config
	spec     *iso8583.MessageSpec
	logger   *log.Logger
	stats    *Stats
	conn     net.Conn
	wg       sync.WaitGroup
	stopChan chan struct{}
}

// Stats tracks simulation statistics
type Stats struct {
	mu            sync.Mutex
	sent          int64
	received      int64
	errors        int64
	startTime     time.Time
	lastSentTime  time.Time
	responseTimes []time.Duration
	tpsCurrent    float64
	tpsAverage    float64
}

// NewSimulator creates a new terminal simulator
func NewSimulator(config Config) *Simulator {
	return &Simulator{
		config:   config,
		spec:     iso8583.Spec87,
		logger:   log.New(os.Stdout, "[TERMINAL] ", log.LstdFlags),
		stats:    &Stats{startTime: time.Now()},
		stopChan: make(chan struct{}),
	}
}

// Run starts the simulator
func (s *Simulator) Run() error {
	s.logger.Printf("Starting terminal simulator")
	s.logger.Printf("Target: %s:%d", s.config.Host, s.config.Port)
	s.logger.Printf("TPS: %d, Count: %d, Duration: %v", s.config.TPS, s.config.Count, s.config.Duration)

	// Connect to switch
	conn, err := net.Dial("tcp", net.JoinHostPort(s.config.Host, fmt.Sprintf("%d", s.config.Port)))
	if err != nil {
		return fmt.Errorf("failed to connect to switch: %w", err)
	}
	defer conn.Close()

	s.conn = conn

	// Start stats reporter
	s.wg.Add(1)
	go s.statsReporter()

	// Start message sender
	s.wg.Add(1)
	go s.messageSender()

	// Start response receiver
	s.wg.Add(1)
	go s.responseReceiver()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		s.logger.Println("Received interrupt signal, stopping...")
	case <-s.stopChan:
		s.logger.Println("Simulation completed")
	}

	// Stop all goroutines
	close(s.stopChan)
	s.wg.Wait()

	// Print final stats
	s.printFinalStats()

	return nil
}
