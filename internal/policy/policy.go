package policy

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jiriki-labs/agentic-wallet/internal/audit"
	"gopkg.in/yaml.v3"
)

// Decision is the result of a policy check.
type Decision string

const (
	DecisionApprove         Decision = "approve"
	DecisionConfirmRequired Decision = "confirm-required"
	DecisionReject          Decision = "reject"
	DecisionDryRun          Decision = "dry-run"
)

// Config holds the policy YAML configuration.
type Config struct {
	AllowedTokens        []string `yaml:"allowedTokens"`
	AllowedChains        []string `yaml:"allowedChains"`
	AllowedMerchants     []string `yaml:"allowedMerchants"`
	MaxAmountPerRequest  string   `yaml:"maxAmountPerRequest"`
	DailyLimit           string   `yaml:"dailyLimit"`
	RequireApprovalAbove string   `yaml:"requireApprovalAbove"`
	Mode                 string   `yaml:"mode"` // "dry-run", "confirm", "auto"
}

// Engine evaluates payment requests against the policy.
type Engine struct {
	cfg Config
	db  *sql.DB
}

// EngineFromConfig creates an Engine directly from a Config and database handle.
// Primarily used by callers (daemon, tests) that have already parsed configuration.
func EngineFromConfig(cfg Config, db *sql.DB) Engine {
	if cfg.Mode == "" {
		cfg.Mode = "confirm"
	}
	return Engine{cfg: cfg, db: db}
}

// Load parses a policy YAML file and returns a new Engine.
func Load(path string, db *sql.DB) (*Engine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read policy file %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse policy YAML: %w", err)
	}
	if cfg.Mode == "" {
		cfg.Mode = "confirm" // default
	}
	return &Engine{cfg: cfg, db: db}, nil
}

// Config returns the loaded configuration (for display).
func (e *Engine) Config() Config {
	return e.cfg
}

// PaymentRequest is the input to the policy check.
type PaymentRequest struct {
	Agent    string
	Merchant string
	Amount   string
	Token    string
	Chain    string
	Reason   string
}

// Result is the output of Decide.
type Result struct {
	Decision  Decision
	Reason    string
	RecordID  int64      // audit row ID inserted in the transaction
	ExpiresAt *time.Time // set for confirm-required
}

// Decide evaluates a payment request within a BEGIN IMMEDIATE transaction.
// It inserts an audit row atomically with the daily-sum check to prevent races.
func (e *Engine) Decide(req PaymentRequest) (Result, error) {
	now := time.Now().UTC()
	todayUTC := now.Format("2006-01-02T00:00:00Z")

	// Use serializable isolation which maps to BEGIN IMMEDIATE in modernc.org/sqlite.
	// Combined with SetMaxOpenConns(1), this ensures only one writer at a time.
	tx, err := e.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Result{}, fmt.Errorf("begin transaction: %w", err)
	}

	var committed bool
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// 1. Mode check
	if e.cfg.Mode == "dry-run" {
		rec := &audit.Record{
			TS: now, Agent: req.Agent, Merchant: req.Merchant,
			Amount: req.Amount, Token: req.Token, Chain: req.Chain,
			Reason: req.Reason, Decision: "dry-run", Status: "dry-run",
		}
		id, err := audit.InsertTx(tx, rec)
		if err != nil {
			return Result{}, err
		}
		if err := tx.Commit(); err != nil {
			return Result{}, err
		}
		committed = true
		return Result{Decision: DecisionDryRun, Reason: "mode is dry-run", RecordID: id}, nil
	}

	// 2. Token check
	if !contains(e.cfg.AllowedTokens, req.Token) {
		rec := &audit.Record{
			TS: now, Agent: req.Agent, Merchant: req.Merchant,
			Amount: req.Amount, Token: req.Token, Chain: req.Chain,
			Reason: req.Reason, Decision: "reject", Status: "rejected",
		}
		id, _ := audit.InsertTx(tx, rec)
		if err := tx.Commit(); err != nil {
			return Result{}, err
		}
		committed = true
		return Result{Decision: DecisionReject, Reason: fmt.Sprintf("token %q not in allowlist", req.Token), RecordID: id}, nil
	}

	// 3. Chain check
	if !contains(e.cfg.AllowedChains, req.Chain) {
		rec := &audit.Record{
			TS: now, Agent: req.Agent, Merchant: req.Merchant,
			Amount: req.Amount, Token: req.Token, Chain: req.Chain,
			Reason: req.Reason, Decision: "reject", Status: "rejected",
		}
		id, _ := audit.InsertTx(tx, rec)
		if err := tx.Commit(); err != nil {
			return Result{}, err
		}
		committed = true
		return Result{Decision: DecisionReject, Reason: fmt.Sprintf("chain %q not in allowlist", req.Chain), RecordID: id}, nil
	}

	// 4. Merchant check
	if !contains(e.cfg.AllowedMerchants, req.Merchant) {
		rec := &audit.Record{
			TS: now, Agent: req.Agent, Merchant: req.Merchant,
			Amount: req.Amount, Token: req.Token, Chain: req.Chain,
			Reason: req.Reason, Decision: "reject", Status: "rejected",
		}
		id, _ := audit.InsertTx(tx, rec)
		if err := tx.Commit(); err != nil {
			return Result{}, err
		}
		committed = true
		return Result{Decision: DecisionReject, Reason: fmt.Sprintf("merchant %q not in allowlist", req.Merchant), RecordID: id}, nil
	}

	// 5. Amount check
	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		return Result{}, fmt.Errorf("parse amount %q: %w", req.Amount, err)
	}

	maxPerReq, err := strconv.ParseFloat(e.cfg.MaxAmountPerRequest, 64)
	if err != nil {
		return Result{}, fmt.Errorf("parse maxAmountPerRequest: %w", err)
	}

	if amount > maxPerReq {
		rec := &audit.Record{
			TS: now, Agent: req.Agent, Merchant: req.Merchant,
			Amount: req.Amount, Token: req.Token, Chain: req.Chain,
			Reason: req.Reason, Decision: "reject", Status: "rejected",
		}
		id, _ := audit.InsertTx(tx, rec)
		if err := tx.Commit(); err != nil {
			return Result{}, err
		}
		committed = true
		return Result{Decision: DecisionReject, Reason: fmt.Sprintf("amount %.2f exceeds maxAmountPerRequest %.2f", amount, maxPerReq), RecordID: id}, nil
	}

	// 6. Daily limit check (within the same transaction)
	dailySum, err := audit.DailySumTx(tx, todayUTC, now)
	if err != nil {
		return Result{}, err
	}

	dailyLimit, err := strconv.ParseFloat(e.cfg.DailyLimit, 64)
	if err != nil {
		return Result{}, fmt.Errorf("parse dailyLimit: %w", err)
	}

	if dailySum+amount > dailyLimit {
		rec := &audit.Record{
			TS: now, Agent: req.Agent, Merchant: req.Merchant,
			Amount: req.Amount, Token: req.Token, Chain: req.Chain,
			Reason: req.Reason, Decision: "reject", Status: "rejected",
		}
		id, _ := audit.InsertTx(tx, rec)
		if err := tx.Commit(); err != nil {
			return Result{}, err
		}
		committed = true
		return Result{
			Decision: DecisionReject,
			Reason:   fmt.Sprintf("daily limit %.2f would be exceeded (current sum: %.2f, request: %.2f)", dailyLimit, dailySum, amount),
			RecordID: id,
		}, nil
	}

	// 7. Determine final decision
	requireAbove, _ := strconv.ParseFloat(e.cfg.RequireApprovalAbove, 64)
	var decision Decision
	var status string
	var expiresAt *time.Time

	if e.cfg.Mode == "confirm" || (e.cfg.RequireApprovalAbove != "" && amount > requireAbove) {
		decision = DecisionConfirmRequired
		status = "confirm-pending"
		exp := now.Add(10 * time.Minute)
		expiresAt = &exp
	} else {
		decision = DecisionApprove
		status = "paid" // will be updated to actual status after payment
	}

	rec := &audit.Record{
		TS: now, Agent: req.Agent, Merchant: req.Merchant,
		Amount: req.Amount, Token: req.Token, Chain: req.Chain,
		Reason: req.Reason, Decision: string(decision), Status: status,
		ExpiresAt: expiresAt,
	}
	id, err := audit.InsertTx(tx, rec)
	if err != nil {
		return Result{}, err
	}

	if err := tx.Commit(); err != nil {
		return Result{}, fmt.Errorf("commit transaction: %w", err)
	}
	committed = true

	return Result{Decision: decision, Reason: "policy approved", RecordID: id, ExpiresAt: expiresAt}, nil
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
