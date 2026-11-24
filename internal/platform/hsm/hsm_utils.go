package hsm

import (
	"fmt"
)

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
