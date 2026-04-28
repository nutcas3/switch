package auth

import (
	"context"
	"time"

	"gopherswitch/internal/modules/auth/domain"
)

// TransactionRepository defines the interface for transaction persistence
type TransactionRepository interface {
	// Create stores a new transaction
	Create(ctx context.Context, txn *domain.Transaction) error

	// Update updates an existing transaction
	Update(ctx context.Context, txn *domain.Transaction) error

	// FindByRRN retrieves a transaction by RRN
	FindByRRN(ctx context.Context, rrn string) (*domain.Transaction, error)

	// FindBySTAN retrieves a transaction by STAN
	FindBySTAN(ctx context.Context, stan string) (*domain.Transaction, error)

	// FindByDateRange retrieves transactions within a date range
	FindByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*domain.Transaction, error)

	// GetDailyTotal gets the total approved amount for a card today
	GetDailyTotal(ctx context.Context, pan string) (int64, error)
}

// CardRepository defines the interface for card operations
type CardRepository interface {
	// FindByPAN retrieves a card by PAN
	FindByPAN(ctx context.Context, pan string) (*domain.Card, error)

	// UpdateStatus updates card status
	UpdateStatus(ctx context.Context, pan string, status string) error
}

// AccountRepository defines the interface for account operations
type AccountRepository interface {
	// FindByCardID retrieves an account by card ID
	FindByCardID(ctx context.Context, cardID int64) (*domain.Account, error)

	// Debit debits an account (returns new balance)
	Debit(ctx context.Context, accountID int64, amount int64) (int64, error)

	// Credit credits an account (returns new balance)
	Credit(ctx context.Context, accountID int64, amount int64) (int64, error)

	// GetBalance retrieves current balance
	GetBalance(ctx context.Context, accountID int64) (int64, error)
}

// HSMAdapter defines the interface for HSM operations
type HSMAdapter interface {
	// VerifyPIN verifies a PIN block
	VerifyPIN(ctx context.Context, pinBlock []byte, pan string) (bool, error)

	// TranslatePIN translates PIN from one key to another
	TranslatePIN(ctx context.Context, pinBlock []byte, fromKey, toKey string) ([]byte, error)

	// GeneratePINBlock generates a PIN block
	GeneratePINBlock(ctx context.Context, pin, pan string) ([]byte, error)

	// EncryptData encrypts sensitive data
	EncryptData(ctx context.Context, data []byte) ([]byte, error)

	// DecryptData decrypts sensitive data
	DecryptData(ctx context.Context, encryptedData []byte) ([]byte, error)
}

// PrivacyService defines the interface for privacy operations
type PrivacyService interface {
	// MaskPAN creates a one-way hash of PAN for storage
	MaskPAN(ctx context.Context, pan string) (string, error)

	// VerifyPAN verifies a PAN against its hash
	VerifyPAN(ctx context.Context, pan, hash string) (bool, error)

	// CreateAuditProof creates a zero-knowledge proof for audit
	CreateAuditProof(ctx context.Context, amount int64) ([]byte, error)

	// VerifyAuditProof verifies an audit proof
	VerifyAuditProof(ctx context.Context, proof []byte, expectedAmount int64) (bool, error)
}

// RoutingService defines the interface for message routing
type RoutingService interface {
	// Route determines the destination for a transaction
	Route(ctx context.Context, pan string) (*domain.RoutingRule, error)

	// SendToIssuer forwards a message to the issuer
	SendToIssuer(ctx context.Context, message []byte, destination *domain.RoutingRule) ([]byte, error)
}

// Logger defines the interface for structured logging
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
}

// MetricsCollector defines the interface for metrics
type MetricsCollector interface {
	// IncrementTransactionCounter increments transaction count
	IncrementTransactionCounter(mti string, responseCode string)

	// ObserveTransactionDuration records transaction duration
	ObserveTransactionDuration(mti string, duration float64)

	// SetActiveConnections sets the active connections gauge
	SetActiveConnections(count int)

	// IncrementErrorCounter increments error count
	IncrementErrorCounter(errorType string)
}
