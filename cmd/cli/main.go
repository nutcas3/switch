package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/moov-io/iso8583"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cli",
	Short: "GopherSwitch CLI utilities",
	Long:  "Command-line interface for GopherSwitch EFT switch operations",
}

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse ISO 8583 messages",
	Long:  "Parse and display ISO 8583 messages in human-readable format",
}

var parseHexCmd = &cobra.Command{
	Use:   "hex --hex <hex-string>",
	Short: "Parse ISO 8583 message from hex string",
	Long:  "Parse an ISO 8583 message from a hex string and display it as JSON",
	Run:   parseHexMessage,
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate test data",
	Long:  "Generate test data for GopherSwitch testing",
}

var generateCardsCmd = &cobra.Command{
	Use:   "cards --count <number>",
	Short: "Generate test card data",
	Long:  "Generate test card records for testing",
	Run:   generateTestCards,
}

var generateMessagesCmd = &cobra.Command{
	Use:   "messages --count <number>",
	Short: "Generate test ISO 8583 messages",
	Long:  "Generate test ISO 8583 messages for load testing",
	Run:   generateTestMessages,
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check transaction status",
	Long:  "Check the status of a transaction by RRN or STAN",
}

var checkRRNCmd = &cobra.Command{
	Use:   "rrn --rrn <retrieval-reference-number>",
	Short: "Check transaction by RRN",
	Long:  "Check transaction status using Retrieval Reference Number",
	Run:   checkTransactionByRRN,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate data",
	Long:  "Validate PAN, PIN, and other financial data",
}

var validatePANCmd = &cobra.Command{
	Use:   "pan --pan <primary-account-number>",
	Short: "Validate PAN format",
	Long:  "Validate Primary Account Number format and checksum",
	Run:   validatePAN,
}

func init() {
	// Parse command
	parseCmd.AddCommand(parseHexCmd)
	parseHexCmd.Flags().String("hex", "", "Hex string of ISO 8583 message")
	parseHexCmd.MarkFlagRequired("hex")

	// Generate command
	generateCmd.AddCommand(generateCardsCmd)
	generateCardsCmd.Flags().Int("count", 10, "Number of cards to generate")

	generateCmd.AddCommand(generateMessagesCmd)
	generateMessagesCmd.Flags().Int("count", 10, "Number of messages to generate")
	generateMessagesCmd.Flags().String("type", "auth", "Message type (auth, reversal, inquiry)")

	// Check command
	checkCmd.AddCommand(checkRRNCmd)
	checkRRNCmd.Flags().String("rrn", "", "Retrieval Reference Number")
	checkRRNCmd.MarkFlagRequired("rrn")

	// Validate command
	validateCmd.AddCommand(validatePANCmd)
	validatePANCmd.Flags().String("pan", "", "Primary Account Number to validate")
	validatePANCmd.MarkFlagRequired("pan")

	// Add all commands to root
	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(validateCmd)
}

func parseHexMessage(cmd *cobra.Command, args []string) {
	hexStr, _ := cmd.Flags().GetString("hex")

	// Remove any whitespace or separators
	hexStr = strings.ReplaceAll(hexStr, " ", "")
	hexStr = strings.ReplaceAll(hexStr, "-", "")
	hexStr = strings.ReplaceAll(hexStr, ":", "")

	// Convert hex to bytes
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		log.Fatalf("Invalid hex string: %v", err)
	}

	// Parse ISO 8583 message
	spec := iso8583.Spec87
	message := iso8583.NewMessage(spec)

	if err := message.Unpack(data); err != nil {
		log.Fatalf("Failed to unpack message: %v", err)
	}

	// Get MTI
	mti, err := message.GetMTI()
	if err != nil {
		log.Fatalf("Failed to get MTI: %v", err)
	}

	// Create result map
	bitmapStr, _ := message.Bitmap().String()
	result := map[string]interface{}{
		"mti":    mti,
		"bitmap": bitmapStr,
		"fields": make(map[string]interface{}),
	}

	// Extract all fields
	fields := message.GetFields()
	for id := range fields {
		value, err := message.GetString(id)
		if err == nil {
			result["fields"].(map[string]interface{})[strconv.Itoa(id)] = map[string]interface{}{
				"value":  value,
				"length": len(value),
			}
		}
	}

	// Output as JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println(string(jsonData))
}

func generateTestCards(cmd *cobra.Command, args []string) {
	count, _ := cmd.Flags().GetInt("count")

	fmt.Printf("Generating %d test cards...\n", count)

	cards := make([]map[string]any, count)

	for i := range count {
		card := generateCard(i)
		cards[i] = card
	}

	// Output as JSON
	jsonData, err := json.MarshalIndent(cards, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println(string(jsonData))
}

func generateCard(index int) map[string]interface{} {
	// Generate different card types
	cardTypes := []string{"VISA", "MASTERCARD", "AMEX", "DISCOVER"}
	cardType := cardTypes[index%len(cardTypes)]

	var pan string
	switch cardType {
	case "VISA":
		pan = fmt.Sprintf("4%d%011d", index, rand.Int()%100000000000)
	case "MASTERCARD":
		pan = fmt.Sprintf("5%d%011d", index, rand.Int()%100000000000)
	case "AMEX":
		pan = fmt.Sprintf("34%d%010d", index, rand.Int()%10000000000)
	case "DISCOVER":
		pan = fmt.Sprintf("6011%d%010d", index, rand.Int()%10000000000)
	}

	// Generate expiration date (MMYY) - 2-5 years from now
	expiry := time.Now().AddDate(2+rand.Intn(4), 0, 0)
	expiryStr := expiry.Format("0106")

	return map[string]interface{}{
		"id":              index + 1,
		"pan":             pan,
		"cardholder_name": fmt.Sprintf("TEST USER %d", index+1),
		"expiration_date": expiryStr,
		"card_type":       cardType,
		"issuer_id":       fmt.Sprintf("%06d", 100000+index),
		"status":          "ACTIVE",
		"daily_limit":     100000000, // $1,000,000 in cents
		"per_txn_limit":   50000000,  // $500,000 in cents
	}
}

func generateTestMessages(cmd *cobra.Command, args []string) {
	count, _ := cmd.Flags().GetInt("count")
	msgType, _ := cmd.Flags().GetString("type")

	fmt.Printf("Generating %d %s messages...\n", count, msgType)

	messages := make([]map[string]interface{}, count)

	for i := 0; i < count; i++ {
		message := generateMessage(i, msgType)
		messages[i] = message
	}

	// Output as JSON
	jsonData, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println(string(jsonData))
}

func generateMessage(index int, msgType string) map[string]interface{} {
	spec := iso8583.Spec87
	message := iso8583.NewMessage(spec)

	// Generate test PAN
	pan := fmt.Sprintf("4%d%011d", index, rand.Int()%100000000000)

	var mti string
	switch msgType {
	case "auth":
		mti = "0100" // Authorization Request
	case "reversal":
		mti = "0400" // Reversal Request
	case "inquiry":
		mti = "0100" // Balance Inquiry (uses same MTI as auth)
	default:
		mti = "0100"
	}

	message.MTI(mti)
	message.Field(2, pan)

	if msgType == "inquiry" {
		message.Field(3, "30") // Balance inquiry processing code
	} else {
		message.Field(3, "00") // Purchase processing code
	}

	// Random amount between $10 and $1000
	amount := 1000 + rand.Int63n(99000) // $10.00 to $999.00 in cents
	message.Field(4, fmt.Sprintf("%012d", amount))

	// Transmission DateTime
	message.Field(7, time.Now().Format("010215304"))

	// STAN
	message.Field(11, fmt.Sprintf("%06d", rand.Intn(999999)))

	// RRN
	message.Field(37, fmt.Sprintf("%012d", rand.Intn(999999999999)))

	// Terminal ID
	message.Field(41, "12345678")

	// Merchant ID
	message.Field(42, "TESTMERCHANT001")

	// Acquiring Institution ID
	message.Field(32, "123456")

	// Currency Code
	message.Field(49, "840") // USD

	// Merchant Type
	message.Field(18, "5999") // Other Services

	// Pack message
	data, err := message.Pack()
	if err != nil {
		log.Fatalf("Failed to pack message: %v", err)
	}

	return map[string]interface{}{
		"index":  index,
		"mti":    mti,
		"type":   msgType,
		"pan":    maskPAN(pan),
		"amount": amount,
		"hex":    hex.EncodeToString(data),
		"length": len(data),
	}
}

func checkTransactionByRRN(cmd *cobra.Command, args []string) {
	rrn, _ := cmd.Flags().GetString("rrn")

	fmt.Printf("Checking transaction with RRN: %s\n", rrn)

	// In a real implementation, this would query the database
	// For now, we'll simulate a response
	transaction := map[string]interface{}{
		"rrn":              rrn,
		"status":           "APPROVED",
		"response_code":    "00",
		"authorization_id": fmt.Sprintf("%06d", rand.Intn(999999)),
		"amount":           50000, // $500.00
		"currency":         "USD",
		"created_at":       time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		"processing_time":  "150ms",
	}

	// Output as JSON
	jsonData, err := json.MarshalIndent(transaction, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println(string(jsonData))
}

func validatePAN(cmd *cobra.Command, args []string) {
	pan, _ := cmd.Flags().GetString("pan")

	fmt.Printf("Validating PAN: %s\n", maskPAN(pan))

	result := map[string]interface{}{
		"pan":       maskPAN(pan),
		"valid":     isValidPAN(pan),
		"length":    len(pan),
		"luhn":      passesLuhn(pan),
		"card_type": getCardType(pan),
	}

	if !result["valid"].(bool) {
		result["error"] = "Invalid PAN format or checksum"
	}

	// Output as JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println(string(jsonData))
}

// Helper functions

func maskPAN(pan string) string {
	if len(pan) < 10 {
		return "****"
	}
	return pan[:6] + "****" + pan[len(pan)-4:]
}

func isValidPAN(pan string) bool {
	if len(pan) < 13 || len(pan) > 19 {
		return false
	}

	for _, char := range pan {
		if char < '0' || char > '9' {
			return false
		}
	}

	return passesLuhn(pan)
}

func passesLuhn(pan string) bool {
	sum := 0
	double := false

	for i := len(pan) - 1; i >= 0; i-- {
		digit := int(pan[i] - '0')
		if digit < 0 || digit > 9 {
			return false
		}

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

func getCardType(pan string) string {
	if len(pan) < 4 {
		return "UNKNOWN"
	}

	bin := pan[:4]
	switch {
	case bin[0] == '4':
		return "VISA"
	case bin[0] == '5' && bin[1] >= '1' && bin[1] <= '5':
		return "MASTERCARD"
	case bin == "3413" || bin == "3713":
		return "AMEX"
	case bin[:2] == "34" || bin[:2] == "37":
		return "AMEX"
	case bin[:4] == "6011" || (bin[0] == '6' && bin[1] == '5' && bin[2] >= '0' && bin[2] <= '9'):
		return "DISCOVER"
	default:
		return "UNKNOWN"
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
