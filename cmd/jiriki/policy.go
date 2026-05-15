package main

import (
	"github.com/spf13/cobra"
)

var (
	policyFileFlag string
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Edit payment policy (interactive TUI)",
	Long:  "Opens a full-screen editor for ~/.config/jiriki/policy.yaml (or under JIRIKI_HOME): limits, allowlists, and mode.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPolicy()
	},
}

func init() {
	policyCmd.Flags().StringVar(&policyFileFlag, "file", "", "policy YAML path (default: config dir policy.yaml)")
	rootCmd.AddCommand(policyCmd)
}

func runPolicy() error {
	return runPolicyTUI()
}
