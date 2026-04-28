package hsm

import (
	"context"
	"fmt"
)

// VerifyPIN verifies a PIN block against the stored PIN
func (h *HSMAdapter) VerifyPIN(ctx context.Context, pinBlock []byte, pan string) (bool, error) {
	if !h.isConnected {
		return false, fmt.Errorf("HSM not connected")
	}

	if len(pinBlock) != 8 {
		return false, fmt.Errorf("invalid PIN block length: expected 8, got %d", len(pinBlock))
	}

	// Build HSM command
	cmd := h.buildCommand("VERIFY_PIN", map[string]string{
		"pin_block": h.bytesToHex(pinBlock),
		"pan":       pan,
	})

	// Send command
	if err := h.sendCommand(cmd); err != nil {
		return false, fmt.Errorf("failed to send PIN verification command: %w", err)
	}

	// Read response
	response, err := h.readResponse()
	if err != nil {
		return false, fmt.Errorf("failed to read PIN verification response: %w", err)
	}

	return response["status"] == "OK" && response["valid"] == "true", nil
}

// TranslatePIN translates a PIN block from terminal key to zone key
func (h *HSMAdapter) TranslatePIN(ctx context.Context, pinBlock []byte, fromKey, toKey string) ([]byte, error) {
	if !h.isConnected {
		return nil, fmt.Errorf("HSM not connected")
	}

	if len(pinBlock) != 8 {
		return nil, fmt.Errorf("invalid PIN block length: expected 8, got %d", len(pinBlock))
	}

	// Build HSM command
	cmd := h.buildCommand("TRANSLATE_PIN", map[string]string{
		"pin_block": h.bytesToHex(pinBlock),
		"from_key":  fromKey,
		"to_key":    toKey,
	})

	// Send command
	if err := h.sendCommand(cmd); err != nil {
		return nil, fmt.Errorf("failed to send PIN translation command: %w", err)
	}

	// Read response
	response, err := h.readResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to read PIN translation response: %w", err)
	}

	if response["status"] != "OK" {
		return nil, fmt.Errorf("PIN translation failed: %s", response["error"])
	}

	translatedBlock := h.hexToBytes(response["translated_pin_block"])
	return translatedBlock, nil
}

// GeneratePINBlock generates a PIN block from PIN and PAN
func (h *HSMAdapter) GeneratePINBlock(ctx context.Context, pin, pan string) ([]byte, error) {
	if !h.isConnected {
		return nil, fmt.Errorf("HSM not connected")
	}

	if len(pin) < 4 || len(pin) > 12 {
		return nil, fmt.Errorf("invalid PIN length: expected 4-12, got %d", len(pin))
	}

	if len(pan) < 12 {
		return nil, fmt.Errorf("invalid PAN length: expected minimum 12, got %d", len(pan))
	}

	// Build HSM command
	cmd := h.buildCommand("GENERATE_PIN_BLOCK", map[string]string{
		"pin": pin,
		"pan": pan,
	})

	// Send command
	if err := h.sendCommand(cmd); err != nil {
		return nil, fmt.Errorf("failed to send PIN block generation command: %w", err)
	}

	// Read response
	response, err := h.readResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to read PIN block generation response: %w", err)
	}

	if response["status"] != "OK" {
		return nil, fmt.Errorf("PIN block generation failed: %s", response["error"])
	}

	pinBlock := h.hexToBytes(response["pin_block"])
	return pinBlock, nil
}
