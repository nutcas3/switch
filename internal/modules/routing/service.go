package routing

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"sync"
	"time"

	"gopherswitch/internal/modules/auth/domain"
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

// LoadRules loads routing rules from database
func (s *Service) LoadRules(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	query := `
		SELECT id, bin_prefix, card_type, destination_host, 
			   destination_port, priority, enabled, created_at, updated_at
		FROM routing_rules WHERE enabled = true
		ORDER BY priority ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to load routing rules: %w", err)
	}
	defer rows.Close()

	s.rules = make(map[string]*domain.RoutingRule)
	for rows.Next() {
		var rule domain.RoutingRule
		err := rows.Scan(
			&rule.ID, &rule.BINPrefix, &rule.CardType, &rule.DestinationHost,
			&rule.DestinationPort, &rule.Priority, &rule.Enabled,
			&rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan routing rule: %w", err)
		}
		s.rules[rule.BINPrefix] = &rule
	}

	s.logger.Info("Loaded routing rules", map[string]interface{}{
		"count": len(s.rules),
	})

	return nil
}

// Route determines the destination for a transaction based on PAN
func (s *Service) Route(ctx context.Context, pan string) (*domain.RoutingRule, error) {
	if len(pan) < 6 {
		return nil, fmt.Errorf("PAN too short for routing (minimum 6 digits)")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Try exact 6-digit BIN match first
	bin6 := pan[:6]
	if rule, exists := s.rules[bin6]; exists {
		s.logger.Debug("Found exact BIN match", map[string]interface{}{
			"bin":  bin6,
			"host": rule.DestinationHost,
			"port": rule.DestinationPort,
		})
		return rule, nil
	}

	// Try 4-digit BIN match
	bin4 := pan[:4]
	for prefix, rule := range s.rules {
		if len(prefix) >= 4 && prefix[:4] == bin4 {
			s.logger.Debug("Found 4-digit BIN match", map[string]interface{}{
				"bin":  bin4,
				"host": rule.DestinationHost,
				"port": rule.DestinationPort,
			})
			return rule, nil
		}
	}

	// Try 1-digit prefix match (card scheme)
	prefix1 := pan[:1]
	switch prefix1 {
	case "4":
		// Default Visa routing
		return &domain.RoutingRule{
			BINPrefix:       "4",
			CardType:        "VISA",
			DestinationHost: "localhost",
			DestinationPort: 9001,
			Priority:        100,
			Enabled:         true,
		}, nil
	case "5":
		// Default Mastercard routing
		return &domain.RoutingRule{
			BINPrefix:       "5",
			CardType:        "MASTERCARD",
			DestinationHost: "localhost",
			DestinationPort: 9002,
			Priority:        100,
			Enabled:         true,
		}, nil
	case "3":
		// Default Amex routing
		return &domain.RoutingRule{
			BINPrefix:       "3",
			CardType:        "AMEX",
			DestinationHost: "localhost",
			DestinationPort: 9003,
			Priority:        100,
			Enabled:         true,
		}, nil
	}

	return nil, fmt.Errorf("no routing rule found for PAN %s", domain.MaskPAN(pan))
}

// SendToIssuer forwards a message to the issuer
func (s *Service) SendToIssuer(ctx context.Context, message []byte, destination *domain.RoutingRule) ([]byte, error) {
	// Create connection key
	connKey := fmt.Sprintf("%s:%d", destination.DestinationHost, destination.DestinationPort)

	// Get or create connection
	conn, err := s.getConnection(connKey, destination.DestinationHost, destination.DestinationPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Set write deadline
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// Send message (add length prefix)
	length := len(message)
	lengthBuf := []byte{byte(length >> 8), byte(length & 0xFF)}
	fullMessage := append(lengthBuf, message...)

	_, err = conn.Write(fullMessage)
	if err != nil {
		s.connMutex.Lock()
		delete(s.connections, connKey)
		s.connMutex.Unlock()
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Read response length
	responseLengthBuf := make([]byte, 2)
	_, err = conn.Read(responseLengthBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read response length: %w", err)
	}

	responseLength := int(responseLengthBuf[0])<<8 | int(responseLengthBuf[1])
	if responseLength <= 0 || responseLength > 8192 {
		return nil, fmt.Errorf("invalid response length: %d", responseLength)
	}

	// Read response
	response := make([]byte, responseLength)
	_, err = conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	s.logger.Debug("Message routed successfully", map[string]interface{}{
		"destination": connKey,
		"request_len": len(message),
		"response_len": len(response),
	})

	return response, nil
}

// getConnection gets or creates a connection to the destination
func (s *Service) getConnection(key, host string, port int) (net.Conn, error) {
	s.connMutex.RLock()
	conn, exists := s.connections[key]
	s.connMutex.RUnlock()

	if exists && !isConnectionClosed(conn) {
		return conn, nil
	}

	// Create new connection
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	// Double-check after acquiring write lock
	if conn, exists := s.connections[key]; exists && !isConnectionClosed(conn) {
		return conn, nil
	}

	address := fmt.Sprintf("%s:%d", host, port)
	newConn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	s.connections[key] = newConn

	s.logger.Info("Created new routing connection", map[string]interface{}{
		"address": address,
		"total_connections": len(s.connections),
	})

	return newConn, nil
}

// isConnectionClosed checks if a connection is closed
func isConnectionClosed(conn net.Conn) bool {
	// Try to set a zero-length deadline to check if connection is alive
	conn.SetReadDeadline(time.Now())
	var buf [1]byte
	_, err := conn.Read(buf[:])
	if err != nil {
		return true
	}
	return false
}

// Close closes all connections
func (s *Service) Close() error {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	var errors []error
	for key, conn := range s.connections {
		if err := conn.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close connection %s: %w", key, err))
		}
		delete(s.connections, key)
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing connections: %v", errors)
	}

	return nil
}

// GetStats returns routing statistics
func (s *Service) GetStats() map[string]interface{} {
	s.mutex.RLock()
	s.connMutex.RLock()
	defer s.mutex.RUnlock()
	defer s.connMutex.RUnlock()

	return map[string]interface{}{
		"routing_rules":    len(s.rules),
		"active_connections": len(s.connections),
	}
}

// AddRule adds a new routing rule
func (s *Service) AddRule(ctx context.Context, rule *domain.RoutingRule) error {
	query := `
		INSERT INTO routing_rules (
			bin_prefix, card_type, destination_host, 
			destination_port, priority, enabled
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err := s.db.QueryRowContext(ctx, query,
		rule.BINPrefix, rule.CardType, rule.DestinationHost,
		rule.DestinationPort, rule.Priority, rule.Enabled,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to add routing rule: %w", err)
	}

	// Update cache
	s.mutex.Lock()
	s.rules[rule.BINPrefix] = rule
	s.mutex.Unlock()

	s.logger.Info("Added routing rule", map[string]interface{}{
		"bin_prefix": rule.BINPrefix,
		"card_type": rule.CardType,
		"destination": fmt.Sprintf("%s:%d", rule.DestinationHost, rule.DestinationPort),
	})

	return nil
}

// RemoveRule removes a routing rule
func (s *Service) RemoveRule(ctx context.Context, binPrefix string) error {
	query := `DELETE FROM routing_rules WHERE bin_prefix = $1`

	result, err := s.db.ExecContext(ctx, query, binPrefix)
	if err != nil {
		return fmt.Errorf("failed to remove routing rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("routing rule with BIN prefix %s not found", binPrefix)
	}

	// Update cache
	s.mutex.Lock()
	delete(s.rules, binPrefix)
	s.mutex.Unlock()

	s.logger.Info("Removed routing rule", map[string]interface{}{
		"bin_prefix": binPrefix,
	})

	return nil
}
