package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/moov-io/iso8583"
	"github.com/spf13/cobra"
)

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

func initGenerateCommand() {
	generateCmd.AddCommand(generateCardsCmd)
	generateCardsCmd.Flags().Int("count", 10, "Number of cards to generate")

	generateCmd.AddCommand(generateMessagesCmd)
	generateMessagesCmd.Flags().Int("count", 10, "Number of messages to generate")
	generateMessagesCmd.Flags().String("type", "auth", "Message type (auth, reversal, inquiry)")
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
