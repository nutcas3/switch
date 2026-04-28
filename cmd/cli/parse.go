package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/moov-io/iso8583"
	"github.com/spf13/cobra"
)

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

func initParseCommand() {
	parseCmd.AddCommand(parseHexCmd)
	parseHexCmd.Flags().String("hex", "", "Hex string of ISO 8583 message")
	parseHexCmd.MarkFlagRequired("hex")
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
