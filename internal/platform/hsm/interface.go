package hsm

import "context"

// HSM defines the interface for HSM operations
type HSM interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	
	// PIN operations
	VerifyPIN(ctx context.Context, pinBlock []byte, pan string) (bool, error)
	TranslatePIN(ctx context.Context, pinBlock []byte, fromKey, toKey string) ([]byte, error)
	GeneratePINBlock(ctx context.Context, pin, pan string) ([]byte, error)
	
	// Data operations
	EncryptData(ctx context.Context, data []byte) ([]byte, error)
	DecryptData(ctx context.Context, encryptedData []byte) ([]byte, error)
	
	// Key management
	AddTerminalKey(ctx context.Context, terminalID string, key []byte) error
	
	// Validation
	ValidatePIN(pin string) error
	ValidatePAN(pan string) error
	
	// Information
	GetKeyInfo() map[string]interface{}
}
