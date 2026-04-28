package main

import (
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cli",
	Short: "GopherSwitch CLI utilities",
	Long:  "Command-line interface for GopherSwitch EFT switch operations",
}

func init() {
	initParseCommand()
	initGenerateCommand()
	initCheckCommand()
	initValidateCommand()

	// Add all commands to root
	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(validateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
