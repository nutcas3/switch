package settlement

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gopherswitch/internal/modules/auth/domain"
)

// GetSettlementBatches retrieves settlement batches within a date range
func (s *Service) GetSettlementBatches(ctx context.Context, startDate, endDate time.Time) ([]*domain.SettlementBatch, error) {
	query := `
		SELECT id, batch_date, total_transactions, total_amount,
			   status, file_path, created_at, completed_at
		FROM settlement_batches 
		WHERE batch_date >= $1 AND batch_date <= $2
		ORDER BY batch_date DESC`

	rows, err := s.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query settlement batches: %w", err)
	}
	defer rows.Close()

	var batches []*domain.SettlementBatch
	for rows.Next() {
		var batch domain.SettlementBatch
		err := rows.Scan(
			&batch.ID, &batch.BatchDate, &batch.TotalTransactions, &batch.TotalAmount,
			&batch.Status, &batch.FilePath, &batch.CreatedAt, &batch.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan settlement batch row: %w", err)
		}
		batches = append(batches, &batch)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settlement batch rows: %w", err)
	}

	return batches, nil
}

// ReconcileBatch performs reconciliation for a settlement batch
func (s *Service) ReconcileBatch(ctx context.Context, batchID int64) error {
	// Get batch details
	var batch domain.SettlementBatch
	query := `SELECT id, batch_date, total_transactions, total_amount, status FROM settlement_batches WHERE id = $1`

	err := s.db.QueryRowContext(ctx, query, batchID).Scan(
		&batch.ID, &batch.BatchDate, &batch.TotalTransactions, &batch.TotalAmount, &batch.Status)
	if err != nil {
		return fmt.Errorf("settlement batch not found: %w", err)
	}

	// Start reconciliation transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin reconciliation transaction: %w", err)
	}
	defer tx.Rollback()

	// Create reconciliation records for all transactions in the batch
	reconQuery := `
		INSERT INTO recon_log (batch_id, transaction_id, matched, discrepancy_amount, discrepancy_reason)
		SELECT $1, id, true, 0, 'MATCHED'
		FROM tran_log 
		WHERE DATE(created_at) = $2 
		AND response_code = '00' 
		AND status = 'APPROVED'`

	result, err := tx.ExecContext(ctx, reconQuery, batchID, batch.BatchDate)
	if err != nil {
		return fmt.Errorf("failed to create reconciliation records: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get reconciliation rows affected: %w", err)
	}

	// Commit reconciliation
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit reconciliation transaction: %w", err)
	}

	s.logger.Info("Batch reconciliation completed", map[string]interface{}{
		"batch_id":         batchID,
		"batch_date":       batch.BatchDate.Format("2006-01-02"),
		"reconciled_count": rowsAffected,
		"expected_count":   batch.TotalTransactions,
	})

	return nil
}

// GetReconciliationReport generates a reconciliation report for a batch
func (s *Service) GetReconciliationReport(ctx context.Context, batchID int64) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_transactions,
			COUNT(CASE WHEN matched = true THEN 1 END) as matched_transactions,
			COUNT(CASE WHEN matched = false THEN 1 END) as unmatched_transactions,
			SUM(CASE WHEN matched = false THEN discrepancy_amount ELSE 0 END) as total_discrepancy
		FROM recon_log WHERE batch_id = $1`

	var totalTxns, matchedTxns, unmatchedTxns, totalDiscrepancy sql.NullInt64
	err := s.db.QueryRowContext(ctx, query, batchID).Scan(
		&totalTxns, &matchedTxns, &unmatchedTxns, &totalDiscrepancy)
	if err != nil {
		return nil, fmt.Errorf("failed to generate reconciliation report: %w", err)
	}

	report := map[string]interface{}{
		"batch_id":              batchID,
		"total_transactions":    totalTxns.Int64,
		"matched_transactions":  matchedTxns.Int64,
		"unmatched_transactions": unmatchedTxns.Int64,
		"total_discrepancy":     totalDiscrepancy.Int64,
	}

	if totalTxns.Valid {
		report["match_rate"] = float64(matchedTxns.Int64) / float64(totalTxns.Int64) * 100
	}

	return report, nil
}
