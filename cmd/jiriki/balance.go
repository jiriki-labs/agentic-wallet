package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Show wallet balance",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBalance()
	},
}

func init() {
	rootCmd.AddCommand(balanceCmd)
}

func runBalance() error {
	fmt.Println("balance: not yet implemented (Phase 6)")
	return nil
}
