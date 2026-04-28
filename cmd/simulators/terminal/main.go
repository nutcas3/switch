package main

import (
	"flag"
	"log"
	"time"
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
