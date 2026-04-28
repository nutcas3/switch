package hsm

import (
	"fmt"
)

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
