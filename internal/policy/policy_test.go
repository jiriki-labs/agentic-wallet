package policy

import (
	"fmt"
	"sync"
	"testing"

	"github.com/jiriki-labs/agentic-wallet/internal/audit"
)

func newTestEngine(t *testing.T, cfg Config) *Engine {
	t.Helper()
	dir := t.TempDir()
	store, err := audit.Open(dir + "/test.db")
	if err != nil {
		t.Fatalf("open audit: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return &Engine{cfg: cfg, db: store.DB()}
}

func defaultCfg() Config {
	return Config{
		AllowedTokens:        []string{"USDC"},
		AllowedChains:        []string{"base-sepolia"},
		AllowedMerchants:     []string{"localhost:4402"},
		MaxAmountPerRequest:  "15.00",
		DailyLimit:           "50.00",
		RequireApprovalAbove: "20.00",
		Mode:                 "auto",
	}
}

func req(amount string) PaymentRequest {
	return PaymentRequest{
		Agent:    "test-agent",
		Merchant: "localhost:4402",
		Amount:   amount,
		Token:    "USDC",
		Chain:    "base-sepolia",
		Reason:   "test",
	}
}

func TestRejectUnknownMerchant(t *testing.T) {
	e := newTestEngine(t, defaultCfg())
	r := req("5.00")
	r.Merchant = "evil.com"
	res, err := e.Decide(r)
	if err != nil {
		t.Fatal(err)
	}
	if res.Decision != DecisionReject {
		t.Fatalf("expected reject, got %v: %s", res.Decision, res.Reason)
	}
}

func TestRejectAmountOverMax(t *testing.T) {
	e := newTestEngine(t, defaultCfg())
	res, err := e.Decide(req("16.00"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Decision != DecisionReject {
		t.Fatalf("expected reject for 16.00 > 15.00, got %v", res.Decision)
	}
}

func TestConfirmRequiredOnThreshold(t *testing.T) {
	cfg := defaultCfg()
	cfg.Mode = "auto"
	cfg.RequireApprovalAbove = "10.00"
	e := newTestEngine(t, cfg)
	res, err := e.Decide(req("12.00"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Decision != DecisionConfirmRequired {
		t.Fatalf("expected confirm-required for 12.00 > 10.00, got %v", res.Decision)
	}
}

func TestDailyLimitAccrual(t *testing.T) {
	cfg := defaultCfg()
	cfg.DailyLimit = "20.00"
	cfg.Mode = "auto"
	e := newTestEngine(t, cfg)

	// First: 10.00 should approve
	r1, err := e.Decide(req("10.00"))
	if err != nil || r1.Decision != DecisionApprove {
		t.Fatalf("first: expected approve, got %v: %v", r1.Decision, err)
	}
	// Second: 10.00 should approve (total 20.00 == limit)
	r2, err := e.Decide(req("10.00"))
	if err != nil || r2.Decision != DecisionApprove {
		t.Fatalf("second: expected approve, got %v: %v", r2.Decision, err)
	}
	// Third: 1.00 should reject (would exceed 20.00)
	r3, err := e.Decide(req("1.00"))
	if err != nil || r3.Decision != DecisionReject {
		t.Fatalf("third: expected reject (over daily limit), got %v: %v", r3.Decision, err)
	}
}

func TestDryRunMode(t *testing.T) {
	cfg := defaultCfg()
	cfg.Mode = "dry-run"
	e := newTestEngine(t, cfg)
	res, err := e.Decide(req("5.00"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Decision != DecisionDryRun {
		t.Fatalf("expected dry-run, got %v", res.Decision)
	}
}

func TestConcurrentRequestsRespectDailyLimit(t *testing.T) {
	// 10 goroutines all try to pay 6.00 USDC against a 30.00 daily limit.
	// Only 5 should be approved (5*6=30). The rest must be rejected.
	const goroutines = 10
	const amount = "6.00"
	const dailyLimit = "30.00" // allows exactly 5

	cfg := defaultCfg()
	cfg.DailyLimit = dailyLimit
	cfg.MaxAmountPerRequest = "10.00"
	cfg.Mode = "auto"
	e := newTestEngine(t, cfg)

	var wg sync.WaitGroup
	results := make([]Result, goroutines)
	errs := make([]error, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			r := req(amount)
			r.Agent = fmt.Sprintf("agent-%d", idx)
			results[idx], errs[idx] = e.Decide(r)
		}(i)
	}
	wg.Wait()

	approvals := 0
	for i, res := range results {
		if errs[i] != nil {
			t.Errorf("goroutine %d error: %v", i, errs[i])
			continue
		}
		if res.Decision == DecisionApprove {
			approvals++
		}
	}

	// Exactly 5 should be approved (30 / 6 = 5)
	if approvals != 5 {
		t.Fatalf("expected exactly 5 approvals, got %d (daily limit: %s, amount: %s)", approvals, dailyLimit, amount)
	}
	t.Logf("Concurrent test: %d approvals, %d rejections out of %d requests", approvals, goroutines-approvals, goroutines)
}
