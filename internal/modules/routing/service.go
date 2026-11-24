package routing

import (
	"database/sql"
	"gopherswitch/internal/modules/auth/domain"
	"net"
	"sync"
)

// Service handles message routing based on BIN patterns
type Service struct {
	db          *sql.DB
	rules       map[string]*domain.RoutingRule
	mutex       sync.RWMutex
	connections map[string]net.Conn
	connMutex   sync.RWMutex
	logger      Logger
}

// Logger interface for structured logging
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Debug(msg string, fields map[string]interface{})
}

// NewService creates a new routing service
func NewService(db *sql.DB, logger Logger) *Service {
	return &Service{
		db:          db,
		rules:       make(map[string]*domain.RoutingRule),
		connections: make(map[string]net.Conn),
		logger:      logger,
	}
}
