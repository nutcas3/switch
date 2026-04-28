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

// Config holds simulator configuration
type Config struct {
	Host        string
	Port        int
	TPS         int
	Count       int
	Duration    time.Duration
	TestPANs    []string
	TestAmounts []int64
}

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

// statsReporter periodically reports statistics
func (s *Simulator) statsReporter() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
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
func (s *Simulator) reportStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	elapsed := time.Since(s.stats.startTime)
	currentTPS := float64(s.stats.sent) / elapsed.Seconds()

	// Calculate average response time
	var avgResponseTime time.Duration
	if len(s.stats.responseTimes) > 0 {
		var total time.Duration
		for _, rt := range s.stats.responseTimes {
			total += rt
		}
		avgResponseTime = total / time.Duration(len(s.stats.responseTimes))
	}

	s.logger.Printf("Stats: Sent=%d, Received=%d, Errors=%d, TPS=%.2f, AvgRT=%v",
		s.stats.sent, s.stats.received, s.stats.errors, currentTPS, avgResponseTime)
}

// printFinalStats prints final simulation statistics
func (s *Simulator) printFinalStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	elapsed := time.Since(s.stats.startTime)
	avgTPS := float64(s.stats.sent) / elapsed.Seconds()

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

	s.logger.Printf("=== Final Statistics ===")
	s.logger.Printf("Duration: %v", elapsed)
	s.logger.Printf("Sent: %d", s.stats.sent)
	s.logger.Printf("Received: %d", s.stats.received)
	s.logger.Printf("Errors: %d", s.stats.errors)
	s.logger.Printf("Average TPS: %.2f", avgTPS)
	s.logger.Printf("Success Rate: %.2f%%", float64(s.stats.received)/float64(s.stats.sent)*100)
	s.logger.Printf("Response Times - Avg: %v, Min: %v, Max: %v", avgResponseTime, minResponseTime, maxResponseTime)
}

func main() {
	// Parse command line flags
	var (
		host     = flag.String("host", "localhost", "Switch host")
		port     = flag.Int("port", 8583, "Switch port")
		tps      = flag.Int("tps", 10, "Transactions per second")
		count    = flag.Int("count", 0, "Total transactions to send (0 = unlimited)")
		duration = flag.Duration("duration", 0, "Duration to run (0 = unlimited)")
	)
	flag.Parse()

	// Default test data
	config := Config{
		Host:     *host,
		Port:     *port,
		TPS:      *tps,
		Count:    *count,
		Duration: *duration,
		TestPANs: []string{
			"4111111111111111", // Visa test
			"5500000000000004", // Mastercard test
			"340000000000009",  // Amex test
		},
		TestAmounts: []int64{
			1000,  // $10.00
			2500,  // $25.00
			5000,  // $50.00
			10000, // $100.00
		},
	}

	// Create and run simulator
	simulator := NewSimulator(config)
	if err := simulator.Run(); err != nil {
		log.Fatalf("Simulator failed: %v", err)
	}
}
