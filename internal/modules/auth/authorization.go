package auth

import (
	"context"
	"strconv"
	"time"

	"gopherswitch/internal/modules/auth/domain"

	"github.com/moov-io/iso8583"
)

// ProcessAuthorization processes an authorization request (0100)
func (s *Service) ProcessAuthorization(ctx context.Context, msg *iso8583.Message) (*iso8583.Message, error) {
	startTime := time.Now()

	// Extract key fields
	pan, err := msg.GetString(2)
	if err != nil {
		return s.createErrorResponse(msg, domain.ResponseCodeInvalidCardNumber)
	}

	amountStr, err := msg.GetString(4)
	if err != nil {
		return s.createErrorResponse(msg, domain.ResponseCodeInvalidAmount)
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return s.createErrorResponse(msg, domain.ResponseCodeInvalidAmount)
	}

	rrn, _ := msg.GetString(37)
	stan, _ := msg.GetString(11)

	s.logger.Info("Processing authorization", map[string]interface{}{
		"pan":    domain.MaskPAN(pan),
		"amount": amount,
		"rrn":    rrn,
		"stan":   stan,
	})

	// Step 1: Validate the card
	card, err := s.cardRepo.FindByPAN(ctx, pan)
	if err != nil {
		s.logger.Error("Card not found", err, map[string]interface{}{"pan": domain.MaskPAN(pan)})
		return s.createErrorResponse(msg, domain.ResponseCodeInvalidCardNumber)
	}

	if card.Status != "ACTIVE" {
		s.logger.Warn("Card not active", map[string]interface{}{
			"pan":    domain.MaskPAN(pan),
			"status": card.Status,
		})
		return s.createErrorResponse(msg, domain.ResponseCodeRestrictedCard)
	}

	// Step 2: Check expiration
	if s.isCardExpired(card.ExpirationDate) {
		return s.createErrorResponse(msg, domain.ResponseCodeExpiredCard)
	}

	// Step 3: Verify PIN if present
	if msg.Bitmap().IsSet(52) {
		pinBlock, _ := msg.GetString(52)
		valid, err := s.hsmAdapter.VerifyPIN(ctx, []byte(pinBlock), pan)
		if err != nil || !valid {
			s.logger.Warn("PIN verification failed", map[string]interface{}{
				"pan": domain.MaskPAN(pan),
			})
			return s.createErrorResponse(msg, domain.ResponseCodeIncorrectPIN)
		}
	}

	// Step 4: Get account and check balance
	account, err := s.accountRepo.FindByCardID(ctx, card.ID)
	if err != nil {
		s.logger.Error("Account not found", err, map[string]interface{}{"card_id": card.ID})
		return s.createErrorResponse(msg, domain.ResponseCodeNoSuchIssuer)
	}

	if account.Status != "ACTIVE" {
		return s.createErrorResponse(msg, domain.ResponseCodeRestrictedCard)
	}

	if account.AvailBalance < amount {
		s.logger.Warn("Insufficient funds", map[string]interface{}{
			"pan":           domain.MaskPAN(pan),
			"amount":        amount,
			"avail_balance": account.AvailBalance,
		})
		return s.createErrorResponse(msg, domain.ResponseCodeInsufficientFunds)
	}

	// Step 5: Check transaction limits
	if amount > card.PerTxnLimit {
		return s.createErrorResponse(msg, domain.ResponseCodeExceedsWithdrawalLimit)
	}

	dailyTotal, err := s.txnRepo.GetDailyTotal(ctx, pan)
	if err == nil && (dailyTotal+amount) > card.DailyLimit {
		return s.createErrorResponse(msg, domain.ResponseCodeExceedsWithdrawalLimit)
	}

	// Step 6: Debit the account
	newBalance, err := s.accountRepo.Debit(ctx, account.ID, amount)
	if err != nil {
		s.logger.Error("Failed to debit account", err, map[string]interface{}{
			"account_id": account.ID,
			"amount":     amount,
		})
		return s.createErrorResponse(msg, domain.ResponseCodeSystemMalfunction)
	}

	// Step 7: Log the transaction with privacy masking
	txn := &domain.Transaction{
		MTI:                  "0100",
		PAN:                  pan,
		ProcessingCode:       s.getFieldOrEmpty(msg, 3),
		Amount:               amount,
		TransmissionDateTime: time.Now(),
		STAN:                 stan,
		RRN:                  rrn,
		ResponseCode:         string(domain.ResponseCodeApproved),
		AuthorizationID:      s.generateAuthID(),
		TerminalID:           s.getFieldOrEmpty(msg, 41),
		CardAcceptorID:       s.getFieldOrEmpty(msg, 42),
		AcquiringInstID:      s.getFieldOrEmpty(msg, 32),
		CurrencyCode:         s.getFieldOrEmpty(msg, 49),
		MerchantType:         s.getFieldOrEmpty(msg, 18),
		Status:               "APPROVED",
		Direction:            "INCOMING",
	}

	// Mask PAN using privacy service
	maskedPAN, err := s.privacySvc.MaskPAN(ctx, pan)
	if err == nil {
		txn.MaskedPAN = maskedPAN
	}

	// Store raw message
	txn.RawMessage, _ = msg.Pack()

	if err := s.txnRepo.Create(ctx, txn); err != nil {
		s.logger.Error("Failed to log transaction", err, map[string]interface{}{
			"rrn": rrn,
		})
	}

	// Step 8: Create approval response
	response := s.createApprovalResponse(msg, txn.AuthorizationID, newBalance)

	// Record metrics
	duration := time.Since(startTime).Seconds()
	s.metrics.ObserveTransactionDuration("0100", duration)
	s.metrics.IncrementTransactionCounter("0100", string(domain.ResponseCodeApproved))

	s.logger.Info("Authorization approved", map[string]interface{}{
		"pan":      domain.MaskPAN(pan),
		"amount":   amount,
		"rrn":      rrn,
		"auth_id":  txn.AuthorizationID,
		"duration": duration,
	})

	return response, nil
}
