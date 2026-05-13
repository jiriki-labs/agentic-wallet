package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var approveCmd = &cobra.Command{
	Use:   "approve",
	Short: "Approve a pending payment",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runApprove(args)
	},
}

func init() {
	rootCmd.AddCommand(approveCmd)
}

func runApprove(args []string) error {
	_ = args
	fmt.Println("approve: not yet implemented (Phase 6)")
	return nil
}
