package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jiriki-labs/agentic-wallet/internal/audit"
	"github.com/jiriki-labs/agentic-wallet/internal/keystore"
	"github.com/jiriki-labs/agentic-wallet/internal/policy"
	"github.com/jiriki-labs/agentic-wallet/internal/x402"
)

// mockX402 records calls and returns a configurable receipt.
type mockX402 struct {
	callCount int32
	lastBody  []byte
	txHash    string
	err       error
}

func (m *mockX402) Pay(_ context.Context, in x402.PayInput) (x402.Receipt, error) {
	atomic.AddInt32(&m.callCount, 1)
	m.lastBody = append([]byte(nil), in.Body...)
	if m.err != nil {
		return x402.Receipt{}, m.err
	}
	return x402.Receipt{TxHash: m.txHash, NonceHex: "aabbcc"}, nil
}

func newTestServer(t *testing.T, policyMode string) (*Server, *audit.Store) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("JIRIKI_HOME", tmp)

	ksStore, err := keystore.New(tmp + "/keystore")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ksStore.Generate("pw"); err != nil {
		t.Fatal(err)
	}
	signer, err := ksStore.Unlock("pw")
	if err != nil {
		t.Fatal(err)
	}

	auditStore, err := audit.Open(tmp + "/audit.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { auditStore.Close() })

	eng := policy.EngineFromConfig(policy.Config{
		AllowedTokens:       []string{"USDC"},
		AllowedChains:       []string{"base-sepolia"},
		AllowedMerchants:    []string{"localhost:4402"},
		MaxAmountPerRequest: "15.00",
		DailyLimit:          "50.00",
		Mode:                policyMode,
	}, auditStore.DB())

	srv := &Server{
		signer:       signer,
		auditStore:   auditStore,
		policyEngine: &eng,
		x402Client:   &mockX402{txHash: "0xTEST"},
		idemCache:    newIdempotencyCache(256, 5*time.Minute),
		confirmQ:     newConfirmQueue(),
	}
	t.Cleanup(func() { srv.idemCache.stop() })
	return srv, auditStore
}

func payRequest(t *testing.T, srv *Server, idemKey string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(payX402Request{
		URL:      "http://localhost:4402/orders",
		Method:   "POST",
		Merchant: "localhost:4402",
		Amount:   "13.42",
		Token:    "USDC",
		Chain:    "base-sepolia",
		Reason:   "test order",
	})
	req := httptest.NewRequest(http.MethodPost, "/pay-x402", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	rec := httptest.NewRecorder()
	srv.handlePayX402(rec, req)
	return rec
}

func TestRejectPathWritesAuditRow(t *testing.T) {
	srv, auditStore := newTestServer(t, "auto")
	body, _ := json.Marshal(payX402Request{
		URL:      "http://evil.com/orders",
		Method:   "POST",
		Merchant: "evil.com", // not in allowlist
		Amount:   "5.00",
		Token:    "USDC",
		Chain:    "base-sepolia",
	})
	req := httptest.NewRequest(http.MethodPost, "/pay-x402", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handlePayX402(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Errorf("expected 402, got %d: %s", rec.Code, rec.Body.String())
	}
	rows, err := auditStore.List("evil.com", time.Time{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("expected audit row for rejected payment, got none")
	}
	if rows[0].Decision != "reject" {
		t.Errorf("expected decision=reject, got %q", rows[0].Decision)
	}
}

func TestDryRunPathWritesAuditRow(t *testing.T) {
	srv, auditStore := newTestServer(t, "dry-run")
	rec := payRequest(t, srv, "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	rows, err := auditStore.List("localhost:4402", time.Time{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("expected audit row for dry-run, got none")
	}
	if rows[0].Status != "dry-run" {
		t.Errorf("expected status=dry-run, got %q", rows[0].Status)
	}
}

func TestConfirmPathReturns202(t *testing.T) {
	srv, _ := newTestServer(t, "confirm")
	rec := payRequest(t, srv, "")
	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp payX402Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.ApprovalID == "" {
		t.Fatal("expected approvalId in response")
	}
	if resp.ExpiresAt == "" {
		t.Fatal("expected expiresAt in response")
	}
	t.Logf("confirm mode returned approvalId=%s", resp.ApprovalID)
}

func TestConfirmThenApproveReturns200(t *testing.T) {
	srv, _ := newTestServer(t, "confirm")
	rec := payRequest(t, srv, "")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}
	var pendingResp payX402Response
	if err := json.Unmarshal(rec.Body.Bytes(), &pendingResp); err != nil {
		t.Fatal(err)
	}

	approveBody, _ := json.Marshal(map[string]string{"approvalId": pendingResp.ApprovalID})
	approveReq := httptest.NewRequest(http.MethodPost, "/approve", bytes.NewReader(approveBody))
	approveReq.Header.Set("Content-Type", "application/json")
	approveRec := httptest.NewRecorder()
	srv.handleApprove(approveRec, approveReq)

	if approveRec.Code != http.StatusOK {
		t.Errorf("expected 200 after approve, got %d: %s", approveRec.Code, approveRec.Body.String())
	}
}

func TestIdempotentRetryReturnsSameResult(t *testing.T) {
	srv, _ := newTestServer(t, "auto")
	mock := srv.x402Client.(*mockX402)

	const key = "test-idem-key-123"
	rec1 := payRequest(t, srv, key)
	rec2 := payRequest(t, srv, key)

	if rec1.Code != rec2.Code {
		t.Errorf("status codes differ: %d vs %d", rec1.Code, rec2.Code)
	}
	if rec1.Body.String() != rec2.Body.String() {
		t.Errorf("bodies differ:\n1: %s\n2: %s", rec1.Body.String(), rec2.Body.String())
	}
	if got := atomic.LoadInt32(&mock.callCount); got != 1 {
		t.Errorf("expected x402 client called once, got %d", got)
	}
	t.Logf("Idempotency test: x402 called %d time(s), both responses identical", atomic.LoadInt32(&mock.callCount))
}

func TestPayX402ForwardsBodyToX402Client(t *testing.T) {
	srv, _ := newTestServer(t, "auto")
	mock := srv.x402Client.(*mockX402)

	orderBody := []byte(`{"dish":"carbonara","servings":2}`)
	payBody, _ := json.Marshal(payX402Request{
		URL:      "http://localhost:4402/orders",
		Method:   "POST",
		Body:     orderBody,
		Merchant: "localhost:4402",
		Amount:   "8.50",
		Token:    "USDC",
		Chain:    "base-sepolia",
		Reason:   "ingredients for carbonara",
		PaymentRequirements: x402.PaymentRequirements{
			Scheme:            "exact",
			Network:           "base-sepolia",
			MaxAmountRequired: "8.50",
			PayTo:             common.HexToAddress("0x1234567890123456789012345678901234567890"),
			Asset:             "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/pay-x402", bytes.NewReader(payBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handlePayX402(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if string(mock.lastBody) != string(orderBody) {
		t.Errorf("x402 client Body: got %q want %q", mock.lastBody, orderBody)
	}
}
