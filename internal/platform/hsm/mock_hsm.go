package hsm

import (
	"context"
	"crypto/des"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// MockHSM implements a mock HSM for testing and development
type MockHSM struct {
	zoneKey   []byte
	termKeys  map[string][]byte
	pinBlocks map[string][]byte // PIN cache for testing
}

// NewMockHSM creates a new mock HSM instance
func NewMockHSM() *MockHSM {
	// Generate mock zone key
	zoneKey := make([]byte, 16) // AES-128 key
	rand.Read(zoneKey)

	return &MockHSM{
		zoneKey:   zoneKey,
		termKeys:  make(map[string][]byte),
		pinBlocks: make(map[string][]byte),
	}
}

// AddTerminalKey adds a terminal key for testing
func (h *MockHSM) AddTerminalKey(terminalID string, key []byte) {
	h.termKeys[terminalID] = key
}

// VerifyPIN verifies a PIN block against the stored PIN
func (h *MockHSM) VerifyPIN(ctx context.Context, pinBlock []byte, pan string) (bool, error) {
	if len(pinBlock) != 8 {
		return false, fmt.Errorf("invalid PIN block length: expected 8, got %d", len(pinBlock))
	}

	// Extract PIN from PIN block (simplified ISO 9564 Format 0)
	pin := h.extractPINFromBlock(pinBlock)
	if pin == "" {
		return false, fmt.Errorf("failed to extract PIN from block")
	}

	// For mock HSM, we'll verify against a simple test PIN hash
	// In production, this would verify against the actual stored PIN
	// Use bcrypt for secure comparison
	err := bcrypt.CompareHashAndPassword([]byte("$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi"), []byte(pin))
	if err != nil {
		return false, nil // PIN doesn't match
	}

	return true, nil
}

// TranslatePIN translates a PIN block from terminal key to zone key
func (h *MockHSM) TranslatePIN(ctx context.Context, pinBlock []byte, fromKey, toKey string) ([]byte, error) {
	if len(pinBlock) != 8 {
		return nil, fmt.Errorf("invalid PIN block length: expected 8, got %d", len(pinBlock))
	}

	// Get source key
	sourceKey, exists := h.termKeys[fromKey]
	if !exists {
		return nil, fmt.Errorf("terminal key not found: %s", fromKey)
	}

	// Decrypt PIN block with source key (simplified)
	decryptedPIN := h.decryptPINBlock(pinBlock, sourceKey)

	// Re-encrypt with destination key (zone key)
	translatedBlock := h.encryptPINBlock(decryptedPIN, h.zoneKey)

	return translatedBlock, nil
}

// GeneratePINBlock generates a PIN block from PIN and PAN
func (h *MockHSM) GeneratePINBlock(ctx context.Context, pin, pan string) ([]byte, error) {
	if len(pin) < 4 || len(pin) > 12 {
		return nil, fmt.Errorf("invalid PIN length: expected 4-12, got %d", len(pin))
	}

	if len(pan) < 12 {
		return nil, fmt.Errorf("invalid PAN length: expected minimum 12, got %d", len(pan))
	}

	// ISO 9564 Format 0 PIN block generation
	pinBlock := h.createPINBlockFormat0(pin, pan)

	return pinBlock, nil
}

// EncryptData encrypts data using DES (simplified)
func (h *MockHSM) EncryptData(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	// Pad data to multiple of 8 bytes
	paddedData := h.padData(data, 8)

	// Create DES cipher
	block, err := des.NewCipher(h.zoneKey[:8]) // Use first 8 bytes for DES
	if err != nil {
		return nil, fmt.Errorf("failed to create DES cipher: %w", err)
	}

	// Encrypt in ECB mode (simplified - use CBC in production)
	encrypted := make([]byte, len(paddedData))
	for i := 0; i < len(paddedData); i += 8 {
		block.Encrypt(encrypted[i:i+8], paddedData[i:i+8])
	}

	return encrypted, nil
}

// DecryptData decrypts data using DES (simplified)
func (h *MockHSM) DecryptData(ctx context.Context, encryptedData []byte) ([]byte, error) {
	if len(encryptedData) == 0 {
		return nil, fmt.Errorf("encrypted data cannot be empty")
	}

	if len(encryptedData)%8 != 0 {
		return nil, fmt.Errorf("encrypted data length must be multiple of 8")
	}

	// Create DES cipher
	block, err := des.NewCipher(h.zoneKey[:8])
	if err != nil {
		return nil, fmt.Errorf("failed to create DES cipher: %w", err)
	}

	// Decrypt in ECB mode (simplified - use CBC in production)
	decrypted := make([]byte, len(encryptedData))
	for i := 0; i < len(encryptedData); i += 8 {
		block.Decrypt(decrypted[i:i+8], encryptedData[i:i+8])
	}

	// Remove padding
	unpaddedData := h.unpadData(decrypted)

	return unpaddedData, nil
}

// Helper methods

// extractPINFromBlock extracts PIN from ISO 9564 Format 0 PIN block
func (h *MockHSM) extractPINFromBlock(pinBlock []byte) string {
	// Simplified PIN extraction from Format 0 block
	// Format: PIN block (8 bytes) = PIN length + PIN + padding
	if len(pinBlock) != 8 {
		return ""
	}

	// First nibble contains PIN length
	pinLength := int(pinBlock[0] >> 4)
	if pinLength < 4 || pinLength > 12 {
		return ""
	}

	// Extract PIN digits (simplified)
	pinDigits := make([]byte, pinLength)
	for i := 0; i < pinLength; i++ {
		if i%2 == 0 {
			pinDigits[i] = (pinBlock[1+i/2] >> 4) + '0'
		} else {
			pinDigits[i] = (pinBlock[1+i/2] & 0x0F) + '0'
		}
	}

	return string(pinDigits)
}

// createPINBlockFormat0 creates ISO 9564 Format 0 PIN block
func (h *MockHSM) createPINBlockFormat0(pin, pan string) []byte {
	pinBlock := make([]byte, 8)

	// First byte: PIN length in first nibble, 0 in second nibble
	pinBlock[0] = byte(len(pin)) << 4

	// Encode PIN digits
	for i := 0; i < len(pin) && i < 12; i++ {
		digit := pin[i] - '0'
		if i%2 == 0 {
			pinBlock[1+i/2] = digit << 4
		} else {
			pinBlock[1+i/2] |= digit
		}
	}

	// Fill remaining with 0xF
	for i := len(pin); i < 12; i++ {
		if i%2 == 0 {
			pinBlock[1+i/2] = 0xF0
		} else {
			pinBlock[1+i/2] |= 0x0F
		}
	}

	return pinBlock
}

// decryptPINBlock decrypts a PIN block (simplified)
func (h *MockHSM) decryptPINBlock(pinBlock, key []byte) []byte {
	// Simplified decryption - just return the block for mock
	// In production, this would use proper cryptographic operations
	return pinBlock
}

// encryptPINBlock encrypts a PIN block (simplified)
func (h *MockHSM) encryptPINBlock(pinData, key []byte) []byte {
	// Simplified encryption - just return the data for mock
	// In production, this would use proper cryptographic operations
	return pinData
}

// padData pads data to specified block size
func (h *MockHSM) padData(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	if padding == 0 {
		padding = blockSize
	}

	padded := make([]byte, len(data)+padding)
	copy(padded, data)

	// PKCS#7 padding
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padding)
	}

	return padded
}

// unpadData removes PKCS#7 padding
func (h *MockHSM) unpadData(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	padding := int(data[len(data)-1])
	if padding > len(data) || padding == 0 {
		return data
	}

	// Verify padding
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return data // Invalid padding
		}
	}

	return data[:len(data)-padding]
}

// GenerateKey generates a random key of specified size
func (h *MockHSM) GenerateKey(size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("key size must be positive")
	}

	key := make([]byte, size)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	return key, nil
}

// ValidatePIN validates PIN format
func (h *MockHSM) ValidatePIN(pin string) error {
	if len(pin) < 4 || len(pin) > 12 {
		return fmt.Errorf("PIN must be 4-12 digits")
	}

	for _, char := range pin {
		if char < '0' || char > '9' {
			return fmt.Errorf("PIN must contain only digits")
		}
	}

	return nil
}

// ValidatePAN validates PAN format
func (h *MockHSM) ValidatePAN(pan string) error {
	if len(pan) < 13 || len(pan) > 19 {
		return fmt.Errorf("PAN must be 13-19 digits")
	}

	for _, char := range pan {
		if char < '0' || char > '9' {
			return fmt.Errorf("PAN must contain only digits")
		}
	}

	return nil
}

// GetKeyInfo returns information about stored keys
func (h *MockHSM) GetKeyInfo() map[string]interface{} {
	return map[string]interface{}{
		"zone_key_size":     len(h.zoneKey),
		"terminal_keys":     len(h.termKeys),
		"cached_pin_blocks": len(h.pinBlocks),
	}
}
