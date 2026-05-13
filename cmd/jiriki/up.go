package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jiriki-labs/agentic-wallet/internal/audit"
	"github.com/jiriki-labs/agentic-wallet/internal/config"
	"github.com/jiriki-labs/agentic-wallet/internal/daemon"
	"github.com/jiriki-labs/agentic-wallet/internal/keystore"
	"github.com/jiriki-labs/agentic-wallet/internal/policy"
	"github.com/spf13/cobra"
)

var (
	upListenTCP      string
	upFacilitatorURL string
	upPolicyFile     string
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the wallet daemon",
	Long:  "Unlocks the keystore, loads policy and audit DB, then serves the local HTTP API (Unix socket by default, optional TCP).",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUp()
	},
}

func init() {
	upCmd.Flags().StringVar(&upListenTCP, "listen-tcp", "", "listen on TCP address (e.g. 127.0.0.1:7402) instead of Unix socket")
	upCmd.Flags().StringVar(&upFacilitatorURL, "facilitator-url", "", "override x402 facilitator URL")
	upCmd.Flags().StringVar(&upPolicyFile, "policy", "", "policy YAML file (default: ~/.config/jiriki/policy.yaml)")
	rootCmd.AddCommand(upCmd)
}

func runUp() error {
	if err := config.CheckAuthPermissions(); err != nil {
		return fmt.Errorf("startup security check failed: %w", err)
	}

	signer, auditStore, engine, err := bootstrapDaemonStack(upPolicyFile)
	if err != nil {
		return err
	}
	defer auditStore.Close()

	if err := ensureBearerTokenForTCP(upListenTCP); err != nil {
		return err
	}

	cfg := daemon.Config{
		ListenTCP:      upListenTCP,
		FacilitatorURL: upFacilitatorURL,
	}
	srv := daemon.New(signer, auditStore, engine, cfg)
	if err := srv.Start(cfg); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

// bootstrapDaemonStack loads keystore, audit DB, and policy engine.
func bootstrapDaemonStack(policyFileFlag string) (*keystore.Signer, *audit.Store, *policy.Engine, error) {
	store, err := keystore.New(config.KeystoreDir())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open keystore: %w", err)
	}
	password, err := keystore.ReadPassword("Keystore password: ")
	if err != nil {
		return nil, nil, nil, err
	}
	signer, err := store.Unlock(password)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unlock keystore: %w", err)
	}
	fmt.Printf("Wallet: %s\n", signer.Address().Hex())

	auditStore, err := audit.Open(config.AuditDB())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open audit DB: %w", err)
	}

	pf := policyFileFlag
	if pf == "" {
		pf = config.PolicyFile()
	}
	engine, err := policy.Load(pf, auditStore.DB())
	if err != nil {
		auditStore.Close()
		return nil, nil, nil, fmt.Errorf("load policy: %w", err)
	}

	return signer, auditStore, engine, nil
}

// ensureBearerTokenForTCP creates ~/.config/jiriki/auth when using TCP and the file is missing.
func ensureBearerTokenForTCP(listenTCP string) error {
	if listenTCP == "" {
		return nil
	}
	if _, err := os.Stat(config.AuthFile()); !os.IsNotExist(err) {
		return err
	}
	if err := config.EnsureDir(); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}
	token := generateDaemonToken()
	if err := config.WriteAuthToken(token); err != nil {
		return fmt.Errorf("write auth token: %w", err)
	}
	fmt.Printf("Generated bearer token at %s\n", config.AuthFile())
	return nil
}

func generateDaemonToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
