package privacy

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service implements privacy-preserving operations
type Service struct {
	auditKey    []byte
	vaultKey    []byte
	salt        []byte
}

// NewService creates a new privacy service
func NewService(auditKey, vaultKey string) *Service {
	return &Service{
		auditKey: []byte(auditKey),
		vaultKey: []byte(vaultKey),
		salt:     []byte("gopherswitch-privacy-salt-2024"),
	}
}

// MaskPAN creates a one-way hash of PAN for storage
func (s *Service) MaskPAN(ctx context.Context, pan string) (string, error) {
	if pan == "" {
		return "", fmt.Errorf("PAN cannot be empty")
	}

	// Create HMAC with audit key
	h := hmac.New(sha256.New, s.auditKey)
	h.Write(s.salt)
	h.Write([]byte(pan))
	hash := h.Sum(nil)

	return hex.EncodeToString(hash), nil
}

// VerifyPAN verifies a PAN against its hash
func (s *Service) VerifyPAN(ctx context.Context, pan, hash string) (bool, error) {
	if pan == "" || hash == "" {
		return false, fmt.Errorf("PAN and hash cannot be empty")
	}

	computedHash, err := s.MaskPAN(ctx, pan)
	if err != nil {
		return false, err
	}

	return computedHash == hash, nil
}

// CreateAuditProof creates a zero-knowledge proof for audit
func (s *Service) CreateAuditProof(ctx context.Context, amount int64) ([]byte, error) {
	if amount < 0 {
		return nil, fmt.Errorf("amount cannot be negative")
	}

	// Simple commitment using HMAC (simplified ZK-proof)
	// In production, use proper ZK-proof libraries
	commitment := fmt.Sprintf("amount:%d:timestamp:%d", amount, time.Now().Unix())
	
	h := hmac.New(sha256.New, s.auditKey)
	h.Write([]byte(commitment))
	proof := h.Sum(nil)

	return proof, nil
}

// VerifyAuditProof verifies an audit proof
func (s *Service) VerifyAuditProof(ctx context.Context, proof []byte, expectedAmount int64) (bool, error) {
	if expectedAmount < 0 {
		return false, fmt.Errorf("expected amount cannot be negative")
	}

	// Recreate commitment (in real ZK-proof, this would be different)
	// This is a simplified verification
	expectedProof, err := s.CreateAuditProof(ctx, expectedAmount)
	if err != nil {
		return false, err
	}

	// Compare proofs (simplified - real ZK-proof would be more complex)
	return hmac.Equal(proof, expectedProof), nil
}

// EncryptSensitiveData encrypts sensitive data using vault key
func (s *Service) EncryptSensitiveData(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	// Simple XOR encryption (use proper encryption in production)
	encrypted := make([]byte, len(data))
	for i, b := range data {
		encrypted[i] = b ^ s.vaultKey[i%len(s.vaultKey)]
	}

	return encrypted, nil
}

// DecryptSensitiveData decrypts sensitive data using vault key
func (s *Service) DecryptSensitiveData(ctx context.Context, encryptedData []byte) ([]byte, error) {
	if len(encryptedData) == 0 {
		return nil, fmt.Errorf("encrypted data cannot be empty")
	}

	// Simple XOR decryption (use proper decryption in production)
	decrypted := make([]byte, len(encryptedData))
	for i, b := range encryptedData {
		decrypted[i] = b ^ s.vaultKey[i%len(s.vaultKey)]
	}

	return decrypted, nil
}

// HashPassword securely hashes a password
func (s *Service) HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func (s *Service) VerifyPassword(password, hash string) (bool, error) {
	if password == "" || hash == "" {
		return false, fmt.Errorf("password and hash cannot be empty")
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, fmt.Errorf("failed to verify password: %w", err)
	}

	return true, nil
}

// MaskAccountNumber masks an account number for logging
func (s *Service) MaskAccountNumber(accountNumber string) string {
	if len(accountNumber) < 8 {
		return "****"
	}
	return accountNumber[:4] + "****" + accountNumber[len(accountNumber)-4:]
}

// GenerateSecureToken generates a secure token for session management
func (s *Service) GenerateSecureToken(ctx context.Context) (string, error) {
	// Generate timestamp-based token
	timestamp := time.Now().UnixNano()
	
	h := hmac.New(sha256.New, s.vaultKey)
	h.Write([]byte(fmt.Sprintf("%d", timestamp)))
	h.Write(s.salt)
	token := h.Sum(nil)

	return hex.EncodeToString(token), nil
}

// ValidateSecureToken validates a secure token
func (s *Service) ValidateSecureToken(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("token cannot be empty")
	}

	// Simple validation - in production, check token expiration and revocation
	_, err := hex.DecodeString(token)
	if err != nil {
		return false, fmt.Errorf("invalid token format: %w", err)
	}

	return true, nil
}

// CreateAuditLog creates a tamper-evident audit log entry
func (s *Service) CreateAuditLog(ctx context.Context, eventType, userID, action, details string) (string, error) {
	timestamp := time.Now().Unix()
	
	// Create audit entry
	entry := fmt.Sprintf("%d:%s:%s:%s:%s", timestamp, eventType, userID, action, details)
	
	// Create HMAC for integrity
	h := hmac.New(sha256.New, s.auditKey)
	h.Write([]byte(entry))
	signature := hex.EncodeToString(h.Sum(nil))
	
	// Combine entry with signature
	auditEntry := entry + ":" + signature
	
	return auditEntry, nil
}

// VerifyAuditLog verifies the integrity of an audit log entry
func (s *Service) VerifyAuditLog(auditEntry string) (bool, error) {
	if auditEntry == "" {
		return false, fmt.Errorf("audit entry cannot be empty")
	}

	// Split entry and signature
	parts := fmt.Sprintf("%s", auditEntry)
	lastColon := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == ':' {
			lastColon = i
			break
		}
	}
	
	if lastColon == -1 {
		return false, fmt.Errorf("invalid audit entry format")
	}

	entry := parts[:lastColon]
	signature := parts[lastColon+1:]

	// Recalculate signature
	h := hmac.New(sha256.New, s.auditKey)
	h.Write([]byte(entry))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return signature == expectedSignature, nil
}
