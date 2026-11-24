package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gopherswitch/internal/modules/auth/domain"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, txn *domain.Transaction) error {
	query := `
		INSERT INTO tran_log (
			mti, pan, masked_pan, processing_code, amount, 
			transmission_datetime, stan, rrn, response_code, 
			authorization_id, terminal_id, card_acceptor_id, 
			acquiring_inst_id, currency_code, merchant_type, 
			status, direction, raw_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		txn.MTI, txn.PAN, txn.MaskedPAN, txn.ProcessingCode, txn.Amount,
		txn.TransmissionDateTime, txn.STAN, txn.RRN, txn.ResponseCode,
		txn.AuthorizationID, txn.TerminalID, txn.CardAcceptorID,
		txn.AcquiringInstID, txn.CurrencyCode, txn.MerchantType,
		txn.Status, txn.Direction, txn.RawMessage,
	).Scan(&txn.ID, &txn.CreatedAt, &txn.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

func (r *TransactionRepository) Update(ctx context.Context, txn *domain.Transaction) error {
	query := `
		UPDATE tran_log SET
			response_code = $1,
			status = $2,
			raw_message = $3,
			updated_at = NOW()
		WHERE id = $4`

	_, err := r.db.ExecContext(ctx, query,
		txn.ResponseCode, txn.Status, txn.RawMessage, txn.ID)

	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

func (r *TransactionRepository) FindByRRN(ctx context.Context, rrn string) (*domain.Transaction, error) {
	query := `
		SELECT id, mti, pan, masked_pan, processing_code, amount,
			   transmission_datetime, stan, rrn, response_code,
			   authorization_id, terminal_id, card_acceptor_id,
			   acquiring_inst_id, currency_code, merchant_type,
			   status, direction, raw_message, created_at, updated_at
		FROM tran_log WHERE rrn = $1`

	var txn domain.Transaction
	err := r.db.QueryRowContext(ctx, query, rrn).Scan(
		&txn.ID, &txn.MTI, &txn.PAN, &txn.MaskedPAN, &txn.ProcessingCode, &txn.Amount,
		&txn.TransmissionDateTime, &txn.STAN, &txn.RRN, &txn.ResponseCode,
		&txn.AuthorizationID, &txn.TerminalID, &txn.CardAcceptorID,
		&txn.AcquiringInstID, &txn.CurrencyCode, &txn.MerchantType,
		&txn.Status, &txn.Direction, &txn.RawMessage, &txn.CreatedAt, &txn.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction with RRN %s not found", rrn)
		}
		return nil, fmt.Errorf("failed to find transaction by RRN: %w", err)
	}

	return &txn, nil
}

func (r *TransactionRepository) FindBySTAN(ctx context.Context, stan string) (*domain.Transaction, error) {
	query := `
		SELECT id, mti, pan, masked_pan, processing_code, amount,
			   transmission_datetime, stan, rrn, response_code,
			   authorization_id, terminal_id, card_acceptor_id,
			   acquiring_inst_id, currency_code, merchant_type,
			   status, direction, raw_message, created_at, updated_at
		FROM tran_log WHERE stan = $1 ORDER BY created_at DESC LIMIT 1`

	var txn domain.Transaction
	err := r.db.QueryRowContext(ctx, query, stan).Scan(
		&txn.ID, &txn.MTI, &txn.PAN, &txn.MaskedPAN, &txn.ProcessingCode, &txn.Amount,
		&txn.TransmissionDateTime, &txn.STAN, &txn.RRN, &txn.ResponseCode,
		&txn.AuthorizationID, &txn.TerminalID, &txn.CardAcceptorID,
		&txn.AcquiringInstID, &txn.CurrencyCode, &txn.MerchantType,
		&txn.Status, &txn.Direction, &txn.RawMessage, &txn.CreatedAt, &txn.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction with STAN %s not found", stan)
		}
		return nil, fmt.Errorf("failed to find transaction by STAN: %w", err)
	}

	return &txn, nil
}

func (r *TransactionRepository) FindByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*domain.Transaction, error) {
	query := `
		SELECT id, mti, pan, masked_pan, processing_code, amount,
			   transmission_datetime, stan, rrn, response_code,
			   authorization_id, terminal_id, card_acceptor_id,
			   acquiring_inst_id, currency_code, merchant_type,
			   status, direction, raw_message, created_at, updated_at
		FROM tran_log 
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions by date range: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		var txn domain.Transaction
		err := rows.Scan(
			&txn.ID, &txn.MTI, &txn.PAN, &txn.MaskedPAN, &txn.ProcessingCode, &txn.Amount,
			&txn.TransmissionDateTime, &txn.STAN, &txn.RRN, &txn.ResponseCode,
			&txn.AuthorizationID, &txn.TerminalID, &txn.CardAcceptorID,
			&txn.AcquiringInstID, &txn.CurrencyCode, &txn.MerchantType,
			&txn.Status, &txn.Direction, &txn.RawMessage, &txn.CreatedAt, &txn.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction row: %w", err)
		}
		transactions = append(transactions, &txn)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows: %w", err)
	}

	return transactions, nil
}

func (r *TransactionRepository) GetDailyTotal(ctx context.Context, pan string) (int64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0) 
		FROM tran_log 
		WHERE masked_pan = $1 
		AND response_code = '00' 
		AND status = 'APPROVED'
		AND DATE(created_at) = CURRENT_DATE`

	var total int64
	err := r.db.QueryRowContext(ctx, query, pan).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get daily total: %w", err)
	}

	return total, nil
}
