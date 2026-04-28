package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gopherswitch/internal/modules/auth/domain"

	"github.com/moov-io/iso8583"
)

// Service implements the authorization business logic
type Service struct {
	txnRepo     TransactionRepository
	cardRepo    CardRepository
	accountRepo AccountRepository
	hsmAdapter  HSMAdapter
	privacySvc  PrivacyService
	routingSvc  RoutingService
	logger      Logger
	metrics     MetricsCollector
}

// NewService creates a new authorization service
func NewService(
	txnRepo TransactionRepository,
	cardRepo CardRepository,
	accountRepo AccountRepository,
	hsmAdapter HSMAdapter,
	privacySvc PrivacyService,
	routingSvc RoutingService,
	logger Logger,
	metrics MetricsCollector,
) *Service {
	return &Service{
		txnRepo:     txnRepo,
		cardRepo:    cardRepo,
		accountRepo: accountRepo,
		hsmAdapter:  hsmAdapter,
		privacySvc:  privacySvc,
		routingSvc:  routingSvc,
		logger:      logger,
		metrics:     metrics,
	}
}

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

// Helper methods

func (s *Service) createErrorResponse(msg *iso8583.Message, responseCode domain.ResponseCode) (*iso8583.Message, error) {
	// Create response message with same spec
	spec := iso8583.Spec87
	response := iso8583.NewMessage(spec)
	response.Field(39, string(responseCode))

	// Copy essential fields
	s.copyFieldIfPresent(msg, response, 2)  // PAN
	s.copyFieldIfPresent(msg, response, 3)  // Processing Code
	s.copyFieldIfPresent(msg, response, 4)  // Amount
	s.copyFieldIfPresent(msg, response, 7)  // Transmission DateTime
	s.copyFieldIfPresent(msg, response, 11) // STAN
	s.copyFieldIfPresent(msg, response, 37) // RRN
	s.copyFieldIfPresent(msg, response, 41) // Terminal ID

	s.metrics.IncrementTransactionCounter("0100", string(responseCode))

	return response, nil
}

func (s *Service) createApprovalResponse(msg *iso8583.Message, authID string, balance int64) *iso8583.Message {
	spec := iso8583.Spec87
	response := iso8583.NewMessage(spec)
	response.Field(39, string(domain.ResponseCodeApproved))

	if authID != "" {
		response.Field(38, authID)
	}

	// Add balance to additional amounts field (54)
	if balance > 0 {
		// Format: Account Type (2) + Amount Type (2) + Currency Code (3) + Amount (12)
		balanceStr := fmt.Sprintf("01%012d", balance)
		response.Field(54, balanceStr)
	}

	// Copy fields from request
	s.copyFieldIfPresent(msg, response, 2)  // PAN
	s.copyFieldIfPresent(msg, response, 3)  // Processing Code
	s.copyFieldIfPresent(msg, response, 4)  // Amount
	s.copyFieldIfPresent(msg, response, 7)  // Transmission DateTime
	s.copyFieldIfPresent(msg, response, 11) // STAN
	s.copyFieldIfPresent(msg, response, 37) // RRN
	s.copyFieldIfPresent(msg, response, 41) // Terminal ID

	return response
}

func (s *Service) copyFieldIfPresent(from, to *iso8583.Message, field int) {
	if from.Bitmap().IsSet(field) {
		value, err := from.GetString(field)
		if err == nil {
			to.Field(field, value)
		}
	}
}

func (s *Service) getFieldOrEmpty(msg *iso8583.Message, field int) string {
	value, err := msg.GetString(field)
	if err != nil {
		return ""
	}
	return value
}

func (s *Service) isCardExpired(expiryDate string) bool {
	if len(expiryDate) != 4 {
		return true
	}

	// Format: YYMM
	now := time.Now()
	currentYY := now.Year() % 100
	currentMM := int(now.Month())

	expiryYY, _ := strconv.Atoi(expiryDate[0:2])
	expiryMM, _ := strconv.Atoi(expiryDate[2:4])

	if expiryYY < currentYY {
		return true
	}
	if expiryYY == currentYY && expiryMM < currentMM {
		return true
	}

	return false
}

func (s *Service) generateAuthID() string {
	// Generate 6-digit authorization ID
	return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
}
