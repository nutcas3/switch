package main

import (
	"strings"
)

// maskPAN masks the PAN by showing only first 6 and last 4 digits
func maskPAN(pan string) string {
	if len(pan) < 10 {
		return "****"
	}
	return pan[:6] + strings.Repeat("*", len(pan)-10) + pan[len(pan)-4:]
}

// isValidPAN validates the basic PAN format
func isValidPAN(pan string) bool {
	if len(pan) < 13 || len(pan) > 19 {
		return false
	}
	for _, c := range pan {
		if c < '0' || c > '9' {
			return false
		}
	}
	return passesLuhn(pan)
}

// passesLuhn validates the PAN using the Luhn algorithm
func passesLuhn(pan string) bool {
	sum := 0
	double := false

	for i := len(pan) - 1; i >= 0; i-- {
		digit := int(pan[i] - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}

	return sum%10 == 0
}

// getCardType determines the card type from the PAN
func getCardType(pan string) string {
	if len(pan) < 1 {
		return "UNKNOWN"
	}

	prefix := pan[0:1]
	if len(pan) >= 2 {
		prefix = pan[0:2]
	}

	switch {
	case strings.HasPrefix(pan, "4"):
		return "VISA"
	case strings.HasPrefix(pan, "5") || strings.HasPrefix(pan, "2"):
		return "MASTERCARD"
	case strings.HasPrefix(pan, "34") || strings.HasPrefix(pan, "37"):
		return "AMEX"
	case strings.HasPrefix(pan, "6011") || strings.HasPrefix(pan, "65"):
		return "DISCOVER"
	case strings.HasPrefix(pan, "35"):
		return "JCB"
	case strings.HasPrefix(pan, "644") || strings.HasPrefix(pan, "645") || strings.HasPrefix(pan, "646") || strings.HasPrefix(pan, "647") || strings.HasPrefix(pan, "648") || strings.HasPrefix(pan, "649"):
		return "DISCOVER"
	default:
		return "UNKNOWN"
	}
}
