package settlement

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopherswitch/internal/modules/auth/domain"
)

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
		"file_path":    filePath,
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
