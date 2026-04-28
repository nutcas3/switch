package main

import (
	"context"
	"fmt"
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
)

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
	hsmAdapter    *hsm.HSMAdapter
	tcpServer     *network.TCPServer
	ctx           context.Context
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
