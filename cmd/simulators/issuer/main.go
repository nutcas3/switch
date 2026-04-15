package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

// statsReporter periodically reports statistics
func (s *IssuerSimulator) statsReporter() {
	defer s.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.reportStats()
		}
	}
}

// reportStats reports current statistics
func (s *IssuerSimulator) reportStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	elapsed := time.Since(s.stats.startTime)
	rate := float64(s.stats.requests) / elapsed.Seconds()

	// Calculate average response time
	var avgResponseTime time.Duration
	if len(s.stats.responseTimes) > 0 {
		var total time.Duration
		for _, rt := range s.stats.responseTimes {
			total += rt
		}
		avgResponseTime = total / time.Duration(len(s.stats.responseTimes))
	}

	approvalRate := float64(0)
	if s.stats.requests > 0 {
		approvalRate = float64(s.stats.approvals) / float64(s.stats.requests) * 100
	}

	s.logger.Printf("Stats: Requests=%d, Approved=%d, Declined=%d, Errors=%d, Rate=%.2f/s, Approval=%.1f%%, AvgRT=%v",
		s.stats.requests, s.stats.approvals, s.stats.declines, s.stats.errors, rate, approvalRate, avgResponseTime)
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

// printFinalStats prints final simulation statistics
func (s *IssuerSimulator) printFinalStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	elapsed := time.Since(s.stats.startTime)
	rate := float64(s.stats.requests) / elapsed.Seconds()

	// Calculate response time statistics
	var avgResponseTime, minResponseTime, maxResponseTime time.Duration
	if len(s.stats.responseTimes) > 0 {
		var total time.Duration
		minResponseTime = s.stats.responseTimes[0]
		maxResponseTime = s.stats.responseTimes[0]

		for _, rt := range s.stats.responseTimes {
			total += rt
			if rt < minResponseTime {
				minResponseTime = rt
			}
			if rt > maxResponseTime {
				maxResponseTime = rt
			}
		}
		avgResponseTime = total / time.Duration(len(s.stats.responseTimes))
	}

	approvalRate := float64(0)
	if s.stats.requests > 0 {
		approvalRate = float64(s.stats.approvals) / float64(s.stats.requests) * 100
	}

	s.logger.Printf("=== Final Statistics ===")
	s.logger.Printf("Duration: %v", elapsed)
	s.logger.Printf("Requests: %d", s.stats.requests)
	s.logger.Printf("Approved: %d", s.stats.approvals)
	s.logger.Printf("Declined: %d", s.stats.declines)
	s.logger.Printf("Errors: %d", s.stats.errors)
	s.logger.Printf("Request Rate: %.2f/s", rate)
	s.logger.Printf("Approval Rate: %.1f%%", approvalRate)
	s.logger.Printf("Response Times - Avg: %v, Min: %v, Max: %v", avgResponseTime, minResponseTime, maxResponseTime)
}

// maskPAN masks a PAN for logging
func maskPAN(pan string) string {
	if len(pan) < 10 {
		return "****"
	}
	return pan[:6] + "****" + pan[len(pan)-4:]
}

func main() {
	// Parse command line flags
	var (
		port         = flag.Int("port", 9001, "Port to listen on")
		approvalRate = flag.Float64("approval-rate", 0.8, "Approval rate (0.0-1.0)")
		minDelay     = flag.Duration("min-delay", 50*time.Millisecond, "Minimum response delay")
		maxDelay     = flag.Duration("max-delay", 200*time.Millisecond, "Maximum response delay")
		balance      = flag.Int64("balance", 100000000, "Account balance in cents ($1,000,000)")
	)
	flag.Parse()

	// Validate configuration
	if *approvalRate < 0 || *approvalRate > 1 {
		log.Fatalf("Approval rate must be between 0.0 and 1.0")
	}

	if *minDelay > *maxDelay {
		log.Fatalf("Minimum delay cannot be greater than maximum delay")
	}

	config := Config{
		Port:         *port,
		ApprovalRate: *approvalRate,
		MinDelay:     *minDelay,
		MaxDelay:     *maxDelay,
		Balance:      *balance,
	}

	// Create and run simulator
	simulator := NewIssuerSimulator(config)

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		simulator.Shutdown()
		os.Exit(0)
	}()

	if err := simulator.Run(); err != nil {
		log.Fatalf("Issuer simulator failed: %v", err)
	}
}
