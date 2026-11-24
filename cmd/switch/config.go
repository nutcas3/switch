package main

import (
	"os"
	"strconv"
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
