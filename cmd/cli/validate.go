package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

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

func initValidateCommand() {
	validateCmd.AddCommand(validatePANCmd)
	validatePANCmd.Flags().String("pan", "", "Primary Account Number to validate")
	validatePANCmd.MarkFlagRequired("pan")
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
