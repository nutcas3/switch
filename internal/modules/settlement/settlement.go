package settlement

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gopherswitch/internal/modules/auth/domain"
)

// ProcessDailySettlement processes end-of-day settlement
func (s *Service) ProcessDailySettlement(ctx context.Context, batchDate time.Time) (*domain.SettlementBatch, error) {
	// Create settlement batch
	batch := &domain.SettlementBatch{
		BatchDate: batchDate,
		Status:    "PROCESSING",
		CreatedAt: time.Now(),
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin settlement transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert batch record
	query := `
		INSERT INTO settlement_batches (batch_date, total_transactions, total_amount, status)
		VALUES ($1, 0, 0, 'PROCESSING')
		RETURNING id`

	err = tx.QueryRowContext(ctx, query, batch.BatchDate).Scan(&batch.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create settlement batch: %w", err)
	}

	// Get approved transactions for the day
	transactions, err := s.getApprovedTransactions(ctx, tx, batchDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get approved transactions: %w", err)
	}

	if len(transactions) == 0 {
		s.logger.Info("No transactions to settle", map[string]interface{}{
			"batch_date": batchDate.Format("2006-01-02"),
		})
		return batch, nil
	}

	// Calculate totals
	var totalAmount int64
	for _, txn := range transactions {
		totalAmount += txn.Amount
	}

	// Update batch totals
	_, err = tx.ExecContext(ctx,
		"UPDATE settlement_batches SET total_transactions = $1, total_amount = $2 WHERE id = $3",
		len(transactions), totalAmount, batch.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update batch totals: %w", err)
	}

	// Generate CSV clearing file
	filePath, err := s.generateClearingFile(ctx, transactions, batchDate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate clearing file: %w", err)
	}

	// Update batch with file path
	_, err = tx.ExecContext(ctx,
		"UPDATE settlement_batches SET file_path = $1, status = 'COMPLETED', completed_at = NOW() WHERE id = $2",
		filePath, batch.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update batch status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit settlement transaction: %w", err)
	}

	// Update batch object
	batch.TotalTransactions = len(transactions)
	batch.TotalAmount = totalAmount
	batch.FilePath = filePath
	batch.Status = "COMPLETED"
	now := time.Now()
	batch.CompletedAt = &now

	s.logger.Info("Settlement completed", map[string]interface{}{
		"batch_id":           batch.ID,
		"batch_date":         batchDate.Format("2006-01-02"),
		"total_transactions": batch.TotalTransactions,
		"total_amount":       batch.TotalAmount,
		"file_path":          batch.FilePath,
	})

	return batch, nil
}

// getApprovedTransactions retrieves approved transactions for a specific date
func (s *Service) getApprovedTransactions(ctx context.Context, tx *sql.Tx, batchDate time.Time) ([]*domain.Transaction, error) {
	query := `
		SELECT id, mti, pan, masked_pan, processing_code, amount,
			   transmission_datetime, stan, rrn, response_code,
			   authorization_id, terminal_id, card_acceptor_id,
			   acquiring_inst_id, currency_code, merchant_type,
			   status, direction, raw_message, created_at, updated_at
		FROM tran_log 
		WHERE DATE(created_at) = $1 
		AND response_code = '00' 
		AND status = 'APPROVED'
		ORDER BY created_at ASC`

	rows, err := tx.QueryContext(ctx, query, batchDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query approved transactions: %w", err)
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
