package auth

import (
	"context"

	"gopherswitch/internal/modules/auth/domain"

	"github.com/moov-io/iso8583"
)

// ProcessReversal processes a reversal request (0400)
func (s *Service) ProcessReversal(ctx context.Context, msg *iso8583.Message) (*iso8583.Message, error) {
	// Extract original RRN from field 90 (Original Data Elements)
	originalData, err := msg.GetString(90)
	if err != nil {
		return s.createErrorResponse(msg, domain.ResponseCodeFormatError)
	}

	// Parse original MTI + STAN + Transmission DateTime + Acquiring Institution ID
	if len(originalData) < 42 {
		return s.createErrorResponse(msg, domain.ResponseCodeFormatError)
	}

	originalRRN, _ := msg.GetString(37)

	// Find original transaction
	originalTxn, err := s.txnRepo.FindByRRN(ctx, originalRRN)
	if err != nil {
		s.logger.Error("Original transaction not found", err, map[string]interface{}{
			"rrn": originalRRN,
		})
		return s.createErrorResponse(msg, domain.ResponseCodeInvalidTransaction)
	}

	// Find account
	card, err := s.cardRepo.FindByPAN(ctx, originalTxn.PAN)
	if err != nil {
		return s.createErrorResponse(msg, domain.ResponseCodeNoSuchIssuer)
	}

	account, err := s.accountRepo.FindByCardID(ctx, card.ID)
	if err != nil {
		return s.createErrorResponse(msg, domain.ResponseCodeNoSuchIssuer)
	}

	// Credit the account
	_, err = s.accountRepo.Credit(ctx, account.ID, originalTxn.Amount)
	if err != nil {
		s.logger.Error("Failed to credit account", err, map[string]interface{}{
			"account_id": account.ID,
			"amount":     originalTxn.Amount,
		})
		return s.createErrorResponse(msg, domain.ResponseCodeSystemMalfunction)
	}

	// Update original transaction status
	originalTxn.Status = "REVERSED"
	s.txnRepo.Update(ctx, originalTxn)

	s.logger.Info("Reversal processed", map[string]interface{}{
		"original_rrn": originalRRN,
		"amount":       originalTxn.Amount,
	})

	// Create approval response
	response := s.createApprovalResponse(msg, string(domain.ResponseCodeApproved), originalTxn.Amount)

	// Copy key fields
	s.copyFieldIfPresent(msg, response, 2)  // PAN
	s.copyFieldIfPresent(msg, response, 3)  // Processing Code
	s.copyFieldIfPresent(msg, response, 4)  // Amount
	s.copyFieldIfPresent(msg, response, 7)  // Transmission DateTime
	s.copyFieldIfPresent(msg, response, 11) // STAN
	s.copyFieldIfPresent(msg, response, 37) // RRN
	s.copyFieldIfPresent(msg, response, 41) // Terminal ID

	return response, nil
}
