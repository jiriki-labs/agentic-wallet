package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jiriki-labs/agentic-wallet/internal/chain"
	"github.com/jiriki-labs/agentic-wallet/internal/config"
	"github.com/jiriki-labs/agentic-wallet/internal/keystore"
	"github.com/jiriki-labs/agentic-wallet/internal/x402"
	"github.com/spf13/cobra"
)

var (
	balanceRPC  string
	balanceUSDC string
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Show wallet on-chain balance",
	Long:  "Queries ETH and USDC balances on Base Sepolia for the address in the local keystore (no daemon required).",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBalance()
	},
}

func init() {
	balanceCmd.Flags().StringVar(&balanceRPC, "rpc", chain.BaseSepoliaRPC, "JSON-RPC URL")
	balanceCmd.Flags().StringVar(&balanceUSDC, "usdc", x402.DefaultUSDCAddress, "USDC contract address")
	rootCmd.AddCommand(balanceCmd)
}

func runBalance() error {
	store, err := keystore.New(config.KeystoreDir())
	if err != nil {
		return fmt.Errorf("open keystore: %w", err)
	}
	addr, err := store.Address()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := chain.New(balanceRPC)
	if err != nil {
		return err
	}
	defer client.Close()

	usdcAddr := common.HexToAddress(balanceUSDC)
	bal, err := client.WalletBalances(ctx, usdcAddr, addr)
	if err != nil {
		return err
	}

	fmt.Printf("Address: %s\n", addr.Hex())
	fmt.Printf("Chain:   base-sepolia\n")
	fmt.Printf("ETH:     %s\n", chain.FormatETH(bal.ETH))
	fmt.Printf("USDC:    %s\n", chain.FormatUSDC(bal.USDC))
	return nil
}
