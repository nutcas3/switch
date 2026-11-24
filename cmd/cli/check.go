package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/spf13/cobra"
)

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

func initCheckCommand() {
	checkCmd.AddCommand(checkRRNCmd)
	checkRRNCmd.Flags().String("rrn", "", "Retrieval Reference Number")
	checkRRNCmd.MarkFlagRequired("rrn")
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
