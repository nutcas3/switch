package routing

import (
	"context"
	"fmt"

	"gopherswitch/internal/modules/auth/domain"
)

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
