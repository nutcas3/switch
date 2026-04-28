package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

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
