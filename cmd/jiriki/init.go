package main

import (
	"fmt"

	"github.com/jiriki-labs/agentic-wallet/internal/config"
	"github.com/jiriki-labs/agentic-wallet/internal/keystore"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a new wallet keystore",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWalletInit()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runWalletInit() error {
	if err := config.EnsureDir(); err != nil {
		return err
	}

	ksDir := config.KeystoreDir()
	store, err := keystore.New(ksDir)
	if err != nil {
		return fmt.Errorf("open keystore: %w", err)
	}

	password, err := keystore.ReadPassword("Enter new keystore password: ")
	if err != nil {
		return err
	}

	addr, err := store.Generate(password)
	if err != nil {
		return err
	}

	fmt.Printf("Wallet created!\n")
	fmt.Printf("Address: %s\n", addr.Hex())
	fmt.Printf("Keystore: %s\n", ksDir)
	fmt.Printf("\nFund this address on base-sepolia with test USDC before running the demo.\n")
	return nil
}
