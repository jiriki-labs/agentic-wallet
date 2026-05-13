package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View payment history",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAudit()
	},
}

func init() {
	rootCmd.AddCommand(auditCmd)
}

func runAudit() error {
	fmt.Println("audit: not yet implemented (Phase 6)")
	return nil
}
