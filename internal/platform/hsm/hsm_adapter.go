package hsm

import (
	"context"
	"fmt"
	"net"
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
	address := net.JoinHostPort(h.host, fmt.Sprintf("%d", h.port))

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
