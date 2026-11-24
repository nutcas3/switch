package hsm

import (
	"fmt"
	"strings"
	"time"
)

// buildCommand builds an HSM command
func (h *HSMAdapter) buildCommand(command string, params map[string]string) string {
	cmd := command
	for key, value := range params {
		cmd += fmt.Sprintf(" %s=%s", key, value)
	}
	return cmd + "\n"
}

// sendCommand sends a command to HSM
func (h *HSMAdapter) sendCommand(command string) error {
	if h.conn == nil {
		return fmt.Errorf("no connection to HSM")
	}

	_, err := h.conn.Write([]byte(command))
	return err
}

// readResponse reads response from HSM
func (h *HSMAdapter) readResponse() (map[string]string, error) {
	if h.conn == nil {
		return nil, fmt.Errorf("no connection to HSM")
	}

	// Set read timeout
	h.conn.SetReadDeadline(time.Now().Add(h.timeout))

	// Read response
	buffer := make([]byte, 1024)
	n, err := h.conn.Read(buffer)
	if err != nil {
		return nil, err
	}

	response := string(buffer[:n])
	return h.parseResponse(response), nil
}

// parseResponse parses HSM response
func (h *HSMAdapter) parseResponse(response string) map[string]string {
	result := make(map[string]string)

	// Simple parsing - split by newlines and then by equals
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		} else {
			result["status"] = line
		}
	}

	return result
}
