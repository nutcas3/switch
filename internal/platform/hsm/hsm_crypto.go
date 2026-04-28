package hsm

import (
	"context"
	"fmt"
)

// EncryptData encrypts data using HSM
func (h *HSMAdapter) EncryptData(ctx context.Context, data []byte) ([]byte, error) {
	if !h.isConnected {
		return nil, fmt.Errorf("HSM not connected")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	// Build HSM command
	cmd := h.buildCommand("ENCRYPT", map[string]string{
		"data": h.bytesToHex(data),
		"key":  "zone_key",
	})

	// Send command
	if err := h.sendCommand(cmd); err != nil {
		return nil, fmt.Errorf("failed to send encryption command: %w", err)
	}

	// Read response
	response, err := h.readResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to read encryption response: %w", err)
	}

	if response["status"] != "OK" {
		return nil, fmt.Errorf("encryption failed: %s", response["error"])
	}

	encrypted := h.hexToBytes(response["encrypted_data"])
	return encrypted, nil
}

// DecryptData decrypts data using HSM
func (h *HSMAdapter) DecryptData(ctx context.Context, encryptedData []byte) ([]byte, error) {
	if !h.isConnected {
		return nil, fmt.Errorf("HSM not connected")
	}

	if len(encryptedData) == 0 {
		return nil, fmt.Errorf("encrypted data cannot be empty")
	}

	// Build HSM command
	cmd := h.buildCommand("DECRYPT", map[string]string{
		"data": h.bytesToHex(encryptedData),
		"key":  "zone_key",
	})

	// Send command
	if err := h.sendCommand(cmd); err != nil {
		return nil, fmt.Errorf("failed to send decryption command: %w", err)
	}

	// Read response
	response, err := h.readResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to read decryption response: %w", err)
	}

	if response["status"] != "OK" {
		return nil, fmt.Errorf("decryption failed: %s", response["error"])
	}

	decrypted := h.hexToBytes(response["decrypted_data"])
	return decrypted, nil
}
