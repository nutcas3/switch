package hsm

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type HSMAdapter struct {
	host         string
	port         int
	timeout      time.Duration
	conn         net.Conn
	isConnected  bool
	sessionKey   []byte
	terminalKeys map[string][]byte
}

type Config struct {
	Host    string
	Port    int
	Timeout time.Duration
	// HSM-specific configuration
	ClientID     string
	ClientSecret string
	ZoneKey      string
}

func NewHSMAdapter(config Config) *HSMAdapter {
	return &HSMAdapter{
		host:         config.Host,
		port:         config.Port,
		timeout:      config.Timeout,
		terminalKeys: make(map[string][]byte),
	}
}

// Connect establishes connection to the HSM
func (h *HSMAdapter) Connect(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", h.host, h.port)

	conn, err := net.DialTimeout("tcp", address, h.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to HSM at %s: %w", address, err)
	}

	h.conn = conn
	h.isConnected = true

	// Perform HSM handshake
	if err := h.performHandshake(ctx); err != nil {
		h.conn.Close()
		h.isConnected = false
		return fmt.Errorf("HSM handshake failed: %w", err)
	}

	return nil
}

// Disconnect closes the HSM connection
func (h *HSMAdapter) Disconnect() error {
	if h.conn != nil {
		h.conn.Close()
	}
	h.isConnected = false
	return nil
}

// performHandshake performs initial HSM handshake
func (h *HSMAdapter) performHandshake(_ context.Context) error {
	// Send login command
	loginCmd := h.buildCommand("LOGIN", map[string]string{
		"client_id":     "gopherswitch",
		"client_secret": "secure_password",
	})

	if err := h.sendCommand(loginCmd); err != nil {
		return fmt.Errorf("failed to send login command: %w", err)
	}

	// Read response
	response, err := h.readResponse()
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	if response["status"] != "OK" {
		return fmt.Errorf("HSM login failed: %s", response["error"])
	}

	// Extract session key
	sessionKeyHex := response["session_key"]
	h.sessionKey = h.hexToBytes(sessionKeyHex)

	return nil
}

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

// GetKeyInfo returns information about stored keys
func (h *HSMAdapter) GetKeyInfo() map[string]interface{} {
	return map[string]interface{}{
		"connected":     h.isConnected,
		"host":          h.host,
		"port":          h.port,
		"terminal_keys": len(h.terminalKeys),
		"session_key":   len(h.sessionKey) > 0,
	}
}

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

// bytesToHex converts bytes to hex string
func (h *HSMAdapter) bytesToHex(data []byte) string {
	return fmt.Sprintf("%x", data)
}

// hexToBytes converts hex string to bytes
func (h *HSMAdapter) hexToBytes(hexStr string) []byte {
	data := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		fmt.Sscanf(hexStr[i:i+2], "%02x", &data[i/2])
	}
	return data
}

// padData pads data to specified block size
func (h *HSMAdapter) padData(data []byte, blockSize int) []byte {
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
func (h *HSMAdapter) unpadData(data []byte) []byte {
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

// ValidatePIN validates PIN format
func (h *HSMAdapter) ValidatePIN(pin string) error {
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
func (h *HSMAdapter) ValidatePAN(pan string) error {
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
