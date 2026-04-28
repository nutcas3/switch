package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run starts the application
func (app *Application) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics server
	go func() {
		app.logger.Info("Starting metrics server", map[string]interface{}{
			"port": app.config.MetricsPort,
		})
		if err := app.metrics.StartMetricsServer(":" + app.config.MetricsPort); err != nil {
			app.logger.Error("Metrics server failed", err, nil)
		}
	}()

	// Start TCP server
	if err := app.tcpServer.Start(":" + app.config.TCPPort); err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}

	app.logger.Info("GopherSwitch started successfully", map[string]interface{}{
		"tcp_port":     app.config.TCPPort,
		"metrics_port": app.config.MetricsPort,
	})

	// Start settlement scheduler
	go app.settlementScheduler(ctx)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	app.logger.Info("Shutdown signal received", nil)

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	return app.shutdown(shutdownCtx)
}

// shutdown performs graceful shutdown
func (app *Application) shutdown(ctx context.Context) error {
	app.logger.Info("Starting graceful shutdown", nil)

	var errors []error

	// Shutdown TCP server
	if err := app.tcpServer.Shutdown(ctx); err != nil {
		errors = append(errors, fmt.Errorf("TCP server shutdown error: %w", err))
	}

	// Close routing connections
	if err := app.routingSvc.Close(); err != nil {
		errors = append(errors, fmt.Errorf("Routing service shutdown error: %w", err))
	}

	// Close database
	if err := app.db.Close(); err != nil {
		errors = append(errors, fmt.Errorf("Database shutdown error: %w", err))
	}

	// Sync logger
	if err := app.logger.Sync(); err != nil {
		errors = append(errors, fmt.Errorf("Logger sync error: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	app.logger.Info("GopherSwitch shutdown complete", nil)
	return nil
}

// settlementScheduler runs settlement processing at scheduled times
func (app *Application) settlementScheduler(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if it's settlement time (default: 2 AM)
			now := time.Now()
			if now.Hour() == 2 && now.Minute() == 0 {
				app.runSettlement(ctx)
			}
		}
	}
}

// runSettlement processes daily settlement
func (app *Application) runSettlement(ctx context.Context) {
	app.logger.Info("Starting daily settlement", nil)

	// Process yesterday's settlement
	yesterday := time.Now().AddDate(0, 0, -1)
	batch, err := app.settlementSvc.ProcessDailySettlement(ctx, yesterday)
	if err != nil {
		app.logger.Error("Settlement processing failed", err, map[string]interface{}{
			"date": yesterday.Format("2006-01-02"),
		})
		return
	}

	// Perform reconciliation
	if batch.ID > 0 {
		if err := app.settlementSvc.ReconcileBatch(ctx, batch.ID); err != nil {
			app.logger.Error("Settlement reconciliation failed", err, map[string]interface{}{
				"batch_id": batch.ID,
			})
		}

		// Update metrics
		app.metrics.IncrementSettlementBatches(batch.Status)
		app.metrics.IncrementSettlementTransactions(batch.BatchDate.Format("2006-01-02"))
		app.metrics.AddSettlementAmount("840", batch.BatchDate.Format("2006-01-02"), float64(batch.TotalAmount))

		app.logger.Info("Settlement completed", map[string]interface{}{
			"batch_id":           batch.ID,
			"total_transactions": batch.TotalTransactions,
			"total_amount":       batch.TotalAmount,
		})
	}
}
