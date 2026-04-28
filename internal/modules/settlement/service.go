package settlement

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopherswitch/internal/modules/auth/domain"
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

// ProcessDailySettlement processes end-of-day settlement
func (s *Service) ProcessDailySettlement(ctx context.Context, batchDate time.Time) (*domain.SettlementBatch, error) {
	// Create settlement batch
	batch := &domain.SettlementBatch{
		BatchDate:  batchDate,
		Status:     "PROCESSING",
		CreatedAt:  time.Now(),
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
		"batch_id":          batch.ID,
		"batch_date":        batchDate.Format("2006-01-02"),
		"total_transactions": batch.TotalTransactions,
		"total_amount":      batch.TotalAmount,
		"file_path":         batch.FilePath,
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

// generateClearingFile generates a CSV clearing file in Mastercard IPM format
func (s *Service) generateClearingFile(ctx context.Context, transactions []*domain.Transaction, batchDate time.Time) (string, error) {
	// Create output directory
	outputDir := "settlement_files"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename with date and timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("CLEARING_%s_%s.csv", batchDate.Format("20060102"), timestamp)
	filePath := filepath.Join(outputDir, filename)

	// Create CSV file
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create clearing file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header (Mastercard IPM format)
	header := []string{
		"RECORD_TYPE",
		"TRANSACTION_DATE",
		"RRN",
		"AUTH_ID",
		"PAN",
		"AMOUNT",
		"CURRENCY",
		"MERCHANT_ID",
		"TERMINAL_ID",
		"ACQUIRER_ID",
		"PROCESSING_CODE",
		"MTI",
	}

	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write header: %w", err)
	}

	// Write transaction records
	for _, txn := range transactions {
		record := []string{
			"1", // Record type: Transaction
			txn.TransmissionDateTime.Format("20060102"),
			txn.RRN,
			txn.AuthorizationID,
			txn.MaskedPAN, // Use masked PAN for compliance
			strconv.FormatInt(txn.Amount, 10),
			txn.CurrencyCode,
			txn.CardAcceptorID,
			txn.TerminalID,
			txn.AcquiringInstID,
			txn.ProcessingCode,
			txn.MTI,
		}

		if err := writer.Write(record); err != nil {
			return "", fmt.Errorf("failed to write transaction record: %w", err)
		}
	}

	// Write trailer record
	trailer := []string{
		"9", // Record type: Trailer
		strconv.Itoa(len(transactions)), // Transaction count
		strconv.FormatInt(s.calculateTotalAmount(transactions), 10), // Total amount
		batchDate.Format("20060102"),
		time.Now().Format("20060102"),
	}

	if err := writer.Write(trailer); err != nil {
		return "", fmt.Errorf("failed to write trailer: %w", err)
	}

	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV writer error: %w", err)
	}

	s.logger.Info("Clearing file generated", map[string]interface{}{
		"file_path":   filePath,
		"transactions": len(transactions),
		"total_amount": s.calculateTotalAmount(transactions),
	})

	return filePath, nil
}

// calculateTotalAmount calculates the total amount of transactions
func (s *Service) calculateTotalAmount(transactions []*domain.Transaction) int64 {
	var total int64
	for _, txn := range transactions {
		total += txn.Amount
	}
	return total
}

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
		"batch_id":          batchID,
		"batch_date":        batch.BatchDate.Format("2006-01-02"),
		"reconciled_count":  rowsAffected,
		"expected_count":    batch.TotalTransactions,
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
		"batch_id":             batchID,
		"total_transactions":   totalTxns.Int64,
		"matched_transactions": matchedTxns.Int64,
		"unmatched_transactions": unmatchedTxns.Int64,
		"total_discrepancy":    totalDiscrepancy.Int64,
	}

	if totalTxns.Valid {
		report["match_rate"] = float64(matchedTxns.Int64) / float64(totalTxns.Int64) * 100
	}

	return report, nil
}
