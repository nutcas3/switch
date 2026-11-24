package database

import (
	"context"
	"database/sql"
	"fmt"

	"gopherswitch/internal/modules/auth/domain"
)

type CardRepository struct {
	db *sql.DB
}

func NewCardRepository(db *sql.DB) *CardRepository {
	return &CardRepository{db: db}
}

func (r *CardRepository) FindByPAN(ctx context.Context, pan string) (*domain.Card, error) {
	query := `
		SELECT id, pan, cardholder_name, expiration_date, card_type,
			   issuer_id, status, daily_limit, per_txn_limit,
			   created_at, updated_at
		FROM cards WHERE pan = $1`

	var card domain.Card
	err := r.db.QueryRowContext(ctx, query, pan).Scan(
		&card.ID, &card.PAN, &card.CardholderName, &card.ExpirationDate, &card.CardType,
		&card.IssuerID, &card.Status, &card.DailyLimit, &card.PerTxnLimit,
		&card.CreatedAt, &card.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("card with PAN %s not found", pan)
		}
		return nil, fmt.Errorf("failed to find card by PAN: %w", err)
	}

	return &card, nil
}

func (r *CardRepository) UpdateStatus(ctx context.Context, pan string, status string) error {
	query := `UPDATE cards SET status = $1, updated_at = NOW() WHERE pan = $2`

	result, err := r.db.ExecContext(ctx, query, status, pan)
	if err != nil {
		return fmt.Errorf("failed to update card status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("card with PAN %s not found", pan)
	}

	return nil
}

func (r *CardRepository) Create(ctx context.Context, card *domain.Card) error {
	query := `
		INSERT INTO cards (
			pan, cardholder_name, expiration_date, card_type,
			issuer_id, status, daily_limit, per_txn_limit
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		card.PAN, card.CardholderName, card.ExpirationDate, card.CardType,
		card.IssuerID, card.Status, card.DailyLimit, card.PerTxnLimit,
	).Scan(&card.ID, &card.CreatedAt, &card.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create card: %w", err)
	}

	return nil
}

func (r *CardRepository) FindByBIN(ctx context.Context, bin string) ([]*domain.Card, error) {
	query := `
		SELECT id, pan, cardholder_name, expiration_date, card_type,
			   issuer_id, status, daily_limit, per_txn_limit,
			   created_at, updated_at
		FROM cards WHERE pan LIKE $1 || '%'`

	rows, err := r.db.QueryContext(ctx, query, bin)
	if err != nil {
		return nil, fmt.Errorf("failed to find cards by BIN: %w", err)
	}
	defer rows.Close()

	var cards []*domain.Card
	for rows.Next() {
		var card domain.Card
		err := rows.Scan(
			&card.ID, &card.PAN, &card.CardholderName, &card.ExpirationDate, &card.CardType,
			&card.IssuerID, &card.Status, &card.DailyLimit, &card.PerTxnLimit,
			&card.CreatedAt, &card.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan card row: %w", err)
		}
		cards = append(cards, &card)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating card rows: %w", err)
	}

	return cards, nil
}
