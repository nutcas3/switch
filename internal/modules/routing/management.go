package routing

import (
	"context"
	"fmt"

	"gopherswitch/internal/modules/auth/domain"
)

// GetStats returns routing statistics
func (s *Service) GetStats() map[string]interface{} {
	s.mutex.RLock()
	s.connMutex.RLock()
	defer s.mutex.RUnlock()
	defer s.connMutex.RUnlock()

	return map[string]interface{}{
		"routing_rules":      len(s.rules),
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
		"bin_prefix":  rule.BINPrefix,
		"card_type":   rule.CardType,
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
