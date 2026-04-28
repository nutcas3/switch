package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"gopherswitch/internal/modules/auth"
	"gopherswitch/internal/modules/privacy"
	"gopherswitch/internal/modules/routing"
	"gopherswitch/internal/modules/settlement"
	"gopherswitch/internal/platform/database"
	"gopherswitch/internal/platform/hsm"
	"gopherswitch/internal/platform/network"
	"gopherswitch/pkg/logger"
	"gopherswitch/pkg/metrics"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Network
	TCPPort string

	// HSM
	AuditKey string
	VaultKey string

	// Logging
	LogLevel  string
	LogFormat string

	// Metrics
	MetricsPort string

	// Settlement
	SettlementTime string
}

// Application holds all application components
type Application struct {
	config        Config
	logger        *logger.Logger
	metrics       *metrics.Collector
	db            *database.PostgreSQL
	authService   *auth.Service
	privacySvc    *privacy.Service
	routingSvc    *routing.Service
	settlementSvc *settlement.Service
	hsmAdapter    hsm.HSM
	tcpServer     *network.TCPServer
	ctx           context.Context
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create context
	ctx := context.Background()

	// Create application
	app, err := NewApplication(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Run application
	if err := app.Run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

// NewApplication creates a new application instance
func NewApplication(ctx context.Context, config Config) (*Application, error) {
	// Create logger
	loggerConfig := logger.Config{
		Level:  config.LogLevel,
		Format: config.LogFormat,
		Output: "stdout",
	}
	log, err := logger.New(loggerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	log.Info("Initializing GopherSwitch", map[string]interface{}{
		"version": "1.0.0",
		"port":    config.TCPPort,
	})

	// Create metrics collector
	metricsCollector := metrics.NewCollector()
	if err := metricsCollector.Register(); err != nil {
		return nil, fmt.Errorf("failed to register metrics: %w", err)
	}

	// Create database connection
	dbConfig := database.Config{
		Host:     config.DBHost,
		Port:     config.DBPort,
		User:     config.DBUser,
		Password: config.DBPassword,
		DBName:   config.DBName,
		SSLMode:  config.DBSSLMode,
	}

	db, err := database.NewPostgreSQL(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Info("Database connected", map[string]interface{}{
		"host":     config.DBHost,
		"port":     config.DBPort,
		"database": config.DBName,
	})

	// Create HSM adapter
	hsmConfig := hsm.Config{
		Host:    getEnv("HSM_HOST", "localhost"),
		Port:    getEnvInt("HSM_PORT", 9999),
		Timeout: 5 * time.Second,
	}
	hsmAdapter := hsm.NewHSMAdapter(hsmConfig)

	// Connect to HSM
	if err := hsmAdapter.Connect(context.Background()); err != nil {
		log.Warn("Failed to connect to HSM, using fallback mode", map[string]interface{}{"error": err})
	} else {
		log.Info("HSM adapter connected", map[string]interface{}{
			"host": hsmConfig.Host,
			"port": hsmConfig.Port,
		})
	}

	// Create privacy service
	privacySvc := privacy.NewService(config.AuditKey, config.VaultKey)
	log.Info("Privacy service initialized", nil)

	// Create repositories
	txnRepo := database.NewTransactionRepository(db.GetDB())
	cardRepo := database.NewCardRepository(db.GetDB())
	accountRepo := database.NewAccountRepository(db.GetDB())

	// Create routing service
	routingSvc := routing.NewService(db.GetDB(), log)
	if err := routingSvc.LoadRules(context.Background()); err != nil {
		log.Warn("Failed to load routing rules", map[string]interface{}{"error": err})
	}
	log.Info("Routing service initialized", nil)

	// Create settlement service
	settlementSvc := settlement.NewService(db.GetDB(), log)
	log.Info("Settlement service initialized", nil)

	// Create auth service
	authService := auth.NewService(
		txnRepo,
		cardRepo,
		accountRepo,
		hsmAdapter,
		privacySvc,
		routingSvc,
		log,
		metricsCollector,
	)
	log.Info("Authorization service initialized", nil)

	// Create TCP server
	tcpServer := network.NewTCPServer(authService, log, metricsCollector)
	log.Info("TCP server created", nil)

	return &Application{
		config:        config,
		logger:        log,
		metrics:       metricsCollector,
		db:            db,
		authService:   authService,
		privacySvc:    privacySvc,
		routingSvc:    routingSvc,
		settlementSvc: settlementSvc,
		hsmAdapter:    hsmAdapter,
		ctx:           ctx,
		tcpServer:     tcpServer,
	}, nil
}

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

// loadConfig loads configuration from environment variables
func loadConfig() (Config, error) {
	config := Config{
		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "gopherswitch"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "gopherswitch"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		// Network
		TCPPort: getEnv("TCP_PORT", "8583"),

		// HSM
		AuditKey: getEnv("AUDIT_KEY", "gopherswitch-audit-key-2024"),
		VaultKey: getEnv("VAULT_KEY", "gopherswitch-vault-key-2024"),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		// Metrics
		MetricsPort: getEnv("METRICS_PORT", "9090"),

		// Settlement
		SettlementTime: getEnv("SETTLEMENT_TIME", "02:00"),
	}

	return config, nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Health check endpoint
func (app *Application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	// Check database health
	if err := app.db.Health(); err != nil {
		health["database"] = "unhealthy"
		health["status"] = "unhealthy"
	} else {
		health["database"] = "healthy"
	}

	// Add stats
	stats := app.tcpServer.GetStats()
	health["tcp_server"] = stats

	routingStats := app.routingSvc.GetStats()
	health["routing"] = routingStats

	hsmStats := app.hsmAdapter.GetKeyInfo()
	health["hsm"] = hsmStats

	w.Header().Set("Content-Type", "application/json")
	if health["status"] == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(health)
}
