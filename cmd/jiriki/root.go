package main

import (
	"os"

	"github.com/spf13/cobra"
)

const cliVersion = "v0.1.0-dev"

var rootCmd = &cobra.Command{
	Use:           "jiriki",
	Short:         "Jiriki is a local wallet daemon CLI for AI agents.",
	SilenceErrors: true,
	SilenceUsage:  true,
	Version:       cliVersion,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() error {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version string",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		_, _ = cmd.OutOrStdout().Write([]byte(cliVersion + "\n"))
	},
}
