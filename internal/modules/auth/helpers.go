package auth

import (
	"fmt"
	"strconv"
	"time"

	"gopherswitch/internal/modules/auth/domain"

	"github.com/moov-io/iso8583"
)

// Helper methods

func (s *Service) createErrorResponse(msg *iso8583.Message, responseCode domain.ResponseCode) (*iso8583.Message, error) {
	// Create response message with same spec
	spec := iso8583.Spec87
	response := iso8583.NewMessage(spec)
	response.Field(39, string(responseCode))

	// Copy essential fields
	s.copyFieldIfPresent(msg, response, 2)  // PAN
	s.copyFieldIfPresent(msg, response, 3)  // Processing Code
	s.copyFieldIfPresent(msg, response, 4)  // Amount
	s.copyFieldIfPresent(msg, response, 7)  // Transmission DateTime
	s.copyFieldIfPresent(msg, response, 11) // STAN
	s.copyFieldIfPresent(msg, response, 37) // RRN
	s.copyFieldIfPresent(msg, response, 41) // Terminal ID

	s.metrics.IncrementTransactionCounter("0100", string(responseCode))

	return response, nil
}

func (s *Service) createApprovalResponse(msg *iso8583.Message, authID string, balance int64) *iso8583.Message {
	spec := iso8583.Spec87
	response := iso8583.NewMessage(spec)
	response.Field(39, string(domain.ResponseCodeApproved))

	if authID != "" {
		response.Field(38, authID)
	}

	// Add balance to additional amounts field (54)
	if balance > 0 {
		// Format: Account Type (2) + Amount Type (2) + Currency Code (3) + Amount (12)
		balanceStr := fmt.Sprintf("01%012d", balance)
		response.Field(54, balanceStr)
	}

	// Copy fields from request
	s.copyFieldIfPresent(msg, response, 2)  // PAN
	s.copyFieldIfPresent(msg, response, 3)  // Processing Code
	s.copyFieldIfPresent(msg, response, 4)  // Amount
	s.copyFieldIfPresent(msg, response, 7)  // Transmission DateTime
	s.copyFieldIfPresent(msg, response, 11) // STAN
	s.copyFieldIfPresent(msg, response, 37) // RRN
	s.copyFieldIfPresent(msg, response, 41) // Terminal ID

	return response
}

func (s *Service) copyFieldIfPresent(from, to *iso8583.Message, field int) {
	if from.Bitmap().IsSet(field) {
		value, err := from.GetString(field)
		if err == nil {
			to.Field(field, value)
		}
	}
}

func (s *Service) getFieldOrEmpty(msg *iso8583.Message, field int) string {
	value, err := msg.GetString(field)
	if err != nil {
		return ""
	}
	return value
}

func (s *Service) isCardExpired(expiryDate string) bool {
	if len(expiryDate) != 4 {
		return true
	}

	// Format: YYMM
	now := time.Now()
	currentYY := now.Year() % 100
	currentMM := int(now.Month())

	expiryYY, _ := strconv.Atoi(expiryDate[0:2])
	expiryMM, _ := strconv.Atoi(expiryDate[2:4])

	if expiryYY < currentYY {
		return true
	}
	if expiryYY == currentYY && expiryMM < currentMM {
		return true
	}

	return false
}

func (s *Service) generateAuthID() string {
	// Generate 6-digit authorization ID
	return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
}
