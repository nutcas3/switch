package hsm

import (
	"context"
	"fmt"
)

// AddTerminalKey adds a terminal key to HSM
func (h *HSMAdapter) AddTerminalKey(ctx context.Context, terminalID string, key []byte) error {
	if !h.isConnected {
		return fmt.Errorf("HSM not connected")
	}

	// Build HSM command
	cmd := h.buildCommand("ADD_TERMINAL_KEY", map[string]string{
		"terminal_id": terminalID,
		"key":         h.bytesToHex(key),
	})

	// Send command
	if err := h.sendCommand(cmd); err != nil {
		return fmt.Errorf("failed to send add terminal key command: %w", err)
	}

	// Read response
	response, err := h.readResponse()
	if err != nil {
		return fmt.Errorf("failed to read add terminal key response: %w", err)
	}

	if response["status"] != "OK" {
		return fmt.Errorf("add terminal key failed: %s", response["error"])
	}

	// Cache the key locally
	h.terminalKeys[terminalID] = key

	return nil
}
