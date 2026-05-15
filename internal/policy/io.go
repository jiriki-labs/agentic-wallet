package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// DefaultConfig returns conservative defaults for a new policy file.
func DefaultConfig() Config {
	return Config{
		AllowedTokens:        []string{"USDC"},
		AllowedChains:        []string{"base-sepolia"},
		AllowedMerchants:     []string{"localhost:4402"},
		MaxAmountPerRequest:  "15.00",
		DailyLimit:           "50.00",
		RequireApprovalAbove: "20.00",
		Mode:                 "confirm",
	}
}

// ReadConfig parses a policy YAML file without opening an audit database.
func ReadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read policy file %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse policy YAML: %w", err)
	}
	if cfg.Mode == "" {
		cfg.Mode = "confirm"
	}
	return cfg, nil
}

// ValidateConfig checks that a config can be used by the policy engine.
func ValidateConfig(cfg Config) error {
	switch cfg.Mode {
	case "", "dry-run", "confirm", "auto":
	default:
		return fmt.Errorf("mode must be dry-run, confirm, or auto (got %q)", cfg.Mode)
	}
	if cfg.MaxAmountPerRequest == "" {
		return fmt.Errorf("maxAmountPerRequest is required")
	}
	if _, err := strconv.ParseFloat(cfg.MaxAmountPerRequest, 64); err != nil {
		return fmt.Errorf("maxAmountPerRequest: %w", err)
	}
	if cfg.DailyLimit == "" {
		return fmt.Errorf("dailyLimit is required")
	}
	if _, err := strconv.ParseFloat(cfg.DailyLimit, 64); err != nil {
		return fmt.Errorf("dailyLimit: %w", err)
	}
	if cfg.RequireApprovalAbove != "" {
		if _, err := strconv.ParseFloat(cfg.RequireApprovalAbove, 64); err != nil {
			return fmt.Errorf("requireApprovalAbove: %w", err)
		}
	}
	return nil
}

// Save writes cfg to path with mode 0600, creating parent directories as needed.
func Save(path string, cfg Config) error {
	if err := ValidateConfig(cfg); err != nil {
		return err
	}
	if cfg.Mode == "" {
		cfg.Mode = "confirm"
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal policy YAML: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create policy dir %s: %w", dir, err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write policy file %s: %w", path, err)
	}
	return nil
}
