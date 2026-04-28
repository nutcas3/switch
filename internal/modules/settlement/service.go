package settlement

import (
	"database/sql"
)

// Service handles end-of-day settlement and reconciliation
type Service struct {
	db     *sql.DB
	logger Logger
}

// Logger interface for structured logging
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Debug(msg string, fields map[string]interface{})
}

// NewService creates a new settlement service
func NewService(db *sql.DB, logger Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}
