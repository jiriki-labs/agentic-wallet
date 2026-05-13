package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Show active policy",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPolicy()
	},
}

func init() {
	rootCmd.AddCommand(policyCmd)
}

func runPolicy() error {
	fmt.Println("policy: not yet implemented (Phase 6)")
	return nil
}
