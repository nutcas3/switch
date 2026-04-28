package domain

import (
	"time"
)

// ResponseCode represents ISO 8583 response codes
type ResponseCode string

const (
	ResponseCodeApproved           ResponseCode = "00"
	ResponseCodeReferToIssuer     ResponseCode = "01"
	ResponseCodeInvalidCardNumber  ResponseCode = "14"
	ResponseCodeInvalidAmount      ResponseCode = "13"
	ResponseCodeIncorrectPIN      ResponseCode = "55"
	ResponseCodeInsufficientFunds ResponseCode = "51"
	ResponseCodeExpiredCard        ResponseCode = "54"
	ResponseCodeRestrictedCard    ResponseCode = "62"
	ResponseCodeSystemMalfunction ResponseCode = "96"
	ResponseCodeNoSuchIssuer      ResponseCode = "91"
	ResponseCodeInvalidTransaction ResponseCode = "30"
	ResponseCodeFormatError       ResponseCode = "30"
	ResponseCodeExceedsWithdrawalLimit ResponseCode = "65"
)

// Transaction represents a financial transaction
type Transaction struct {
	ID                   int64     `json:"id"`
	MTI                  string    `json:"mti"`
	PAN                  string    `json:"pan"`
	MaskedPAN            string    `json:"masked_pan"`
	ProcessingCode       string    `json:"processing_code"`
	Amount               int64     `json:"amount"`
	TransmissionDateTime time.Time `json:"transmission_datetime"`
	STAN                 string    `json:"stan"`
	RRN                  string    `json:"rrn"`
	ResponseCode         string    `json:"response_code"`
	AuthorizationID      string    `json:"authorization_id"`
	TerminalID           string    `json:"terminal_id"`
	CardAcceptorID       string    `json:"card_acceptor_id"`
	AcquiringInstID      string    `json:"acquiring_inst_id"`
	CurrencyCode         string    `json:"currency_code"`
	MerchantType         string    `json:"merchant_type"`
	Status               string    `json:"status"`
	Direction            string    `json:"direction"`
	RawMessage           []byte    `json:"raw_message"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Card represents a payment card
type Card struct {
	ID            int64  `json:"id"`
	PAN           string `json:"pan"`
	CardholderName string `json:"cardholder_name"`
	ExpirationDate string `json:"expiration_date"`
	CardType      string `json:"card_type"`
	IssuerID      string `json:"issuer_id"`
	Status        string `json:"status"`
	DailyLimit    int64  `json:"daily_limit"`
	PerTxnLimit   int64  `json:"per_txn_limit"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Account represents a bank account
type Account struct {
	ID           int64  `json:"id"`
	AccountNumber string `json:"account_number"`
	CardID       int64  `json:"card_id"`
	Balance      int64  `json:"balance"`
	AvailBalance int64  `json:"avail_balance"`
	CurrencyCode string `json:"currency_code"`
	Status       string `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RoutingRule represents a routing configuration
type RoutingRule struct {
	ID              int64  `json:"id"`
	BINPrefix       string `json:"bin_prefix"`
	CardType        string `json:"card_type"`
	DestinationHost string `json:"destination_host"`
	DestinationPort int    `json:"destination_port"`
	Priority        int    `json:"priority"`
	Enabled         bool   `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SettlementBatch represents a daily settlement batch
type SettlementBatch struct {
	ID               int64      `json:"id"`
	BatchDate        time.Time  `json:"batch_date"`
	TotalTransactions int       `json:"total_transactions"`
	TotalAmount      int64      `json:"total_amount"`
	Status           string     `json:"status"`
	FilePath         string     `json:"file_path"`
	CreatedAt        time.Time  `json:"created_at"`
	CompletedAt      *time.Time `json:"completed_at"`
}

// AuditLog represents an audit trail entry
type AuditLog struct {
	ID            int64      `json:"id"`
	TransactionID *int64     `json:"transaction_id"`
	EventType     string     `json:"event_type"`
	UserID        string     `json:"user_id"`
	IPAddress     string     `json:"ip_address"`
	Action        string     `json:"action"`
	Details       string     `json:"details"`
	Timestamp     time.Time  `json:"timestamp"`
}

// ReconciliationLog represents settlement reconciliation
type ReconciliationLog struct {
	ID               int64       `json:"id"`
	BatchID          int64       `json:"batch_id"`
	TransactionID    int64       `json:"transaction_id"`
	Matched          bool        `json:"matched"`
	DiscrepancyAmount int64      `json:"discrepancy_amount"`
	DiscrepancyReason string      `json:"discrepancy_reason"`
	Resolved         bool        `json:"resolved"`
	CreatedAt        time.Time   `json:"created_at"`
	ResolvedAt       *time.Time  `json:"resolved_at"`
}

// MaskPAN masks a PAN for logging (show first 6 and last 4 digits)
func MaskPAN(pan string) string {
	if len(pan) < 10 {
		return "****"
	}
	return pan[:6] + "****" + pan[len(pan)-4:]
}

// IsValidPAN checks if a PAN is valid (Luhn algorithm)
func IsValidPAN(pan string) bool {
	if len(pan) < 13 || len(pan) > 19 {
		return false
	}

	sum := 0
	double := false

	for i := len(pan) - 1; i >= 0; i-- {
		digit := int(pan[i] - '0')
		if digit < 0 || digit > 9 {
			return false
		}

		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		double = !double
	}

	return sum%10 == 0
}

// GetCardType returns card type based on BIN
func GetCardType(pan string) string {
	if len(pan) < 4 {
		return "UNKNOWN"
	}

	bin := pan[:4]
	switch {
	case bin[0] == '4':
		return "VISA"
	case bin[0] == '5' && bin[1] >= '1' && bin[1] <= '5':
		return "MASTERCARD"
	case bin == "3413" || bin == "3713":
		return "AMEX"
	case bin[:2] == "34" || bin[:2] == "37":
		return "AMEX"
	case bin[:4] == "6011" || (bin[0] == '6' && bin[1] == '5' && bin[2] >= '0' && bin[2] <= '9'):
		return "DISCOVER"
	default:
		return "UNKNOWN"
	}
}
