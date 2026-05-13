package x402

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jiriki-labs/agentic-wallet/internal/keystore"
	fxevm "github.com/x402-foundation/x402/go/mechanisms/evm"
	fxv1 "github.com/x402-foundation/x402/go/mechanisms/evm/v1"
)

// keystoreSigner adapts keystore.Signer to x402.Signer
type keystoreSigner struct {
	s *keystore.Signer
}

func (k *keystoreSigner) Address() common.Address           { return k.s.Address() }
func (k *keystoreSigner) SignHash(h []byte) ([]byte, error) { return k.s.SignHash(h) }

func newTestSigner(t *testing.T) *keystoreSigner {
	t.Helper()
	dir := t.TempDir()
	store, err := keystore.New(dir)
	if err != nil {
		t.Fatalf("keystore.New: %v", err)
	}
	if _, err = store.Generate("test-pw"); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	signer, err := store.Unlock("test-pw")
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	return &keystoreSigner{s: signer}
}

func TestNonceUniquenessAcrossRequests(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		n, err := fxevm.CreateNonce()
		if err != nil {
			t.Fatalf("CreateNonce [%d]: %v", i, err)
		}
		if _, dup := seen[n]; dup {
			t.Fatalf("duplicate nonce at iteration %d: %s", i, n)
		}
		seen[n] = struct{}{}
	}
	t.Logf("TestNonceUniquenessAcrossRequests: 1000 unique nonces verified")
}

func TestExpiredAuthorizationRefused(t *testing.T) {
	expiredBefore := time.Now().Unix() - 1
	validAfter := time.Now().Unix() - 10

	err := ValidateWindow(validAfter, expiredBefore)
	if err == nil {
		t.Fatal("expected error for expired validBefore, got nil")
	}
	t.Logf("Correctly refused: %v", err)
}

func TestValidateWindowTooLarge(t *testing.T) {
	now := time.Now().Unix()
	err := ValidateWindow(now-5, now+3601)
	if err == nil {
		t.Fatal("expected error for >3600s window, got nil")
	}
	t.Logf("Correctly refused large window: %v", err)
}

func TestValidateWindowOK(t *testing.T) {
	now := time.Now().Unix()
	err := ValidateWindow(now-5, now+300)
	if err != nil {
		t.Fatalf("expected ok for 305s window, got: %v", err)
	}
}

// strengthenedMockFacilitator decodes the facilitator wire format, verifies the
// EIP-3009 digest with the same rules as the in-tree test used to enforce.
func strengthenedMockFacilitator(t *testing.T, expectedAddr common.Address, expectedValue *big.Int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/settle" {
			http.Error(w, "not found", 404)
			return
		}

		var envelope struct {
			PaymentPayload struct {
				Payload struct {
					Signature     string `json:"signature"`
					Authorization struct {
						From        string `json:"from"`
						To          string `json:"to"`
						Value       string `json:"value"`
						ValidAfter  string `json:"validAfter"`
						ValidBefore string `json:"validBefore"`
						Nonce       string `json:"nonce"`
					} `json:"authorization"`
				} `json:"payload"`
			} `json:"paymentPayload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Errorf("mock facilitator: decode body: %v", err)
			http.Error(w, "bad request", 400)
			return
		}

		auth := envelope.PaymentPayload.Payload.Authorization
		sig := envelope.PaymentPayload.Payload.Signature

		nonceHex := strings.TrimPrefix(auth.Nonce, "0x")
		nonceBytes, err := hex.DecodeString(nonceHex)
		if err != nil || len(nonceBytes) != 32 {
			t.Errorf("mock facilitator: nonce must be 32 bytes, got %d: %v", len(nonceBytes), err)
		}

		var validBefore big.Int
		validBefore.SetString(auth.ValidBefore, 10)
		now := time.Now().Unix()
		if validBefore.Int64() <= now {
			t.Errorf("mock facilitator: validBefore %d is not in the future (now: %d)", validBefore.Int64(), now)
		}

		var value big.Int
		value.SetString(auth.Value, 10)
		if value.Cmp(expectedValue) != 0 {
			t.Errorf("mock facilitator: value %s != expected %s", value.String(), expectedValue.String())
		}

		chainID, err := fxv1.GetEvmChainId("base-sepolia")
		if err != nil {
			t.Errorf("mock facilitator: chain id: %v", err)
			http.Error(w, "bad config", 500)
			return
		}
		ai, err := fxv1.GetAssetInfo("base-sepolia", DefaultUSDCAddress)
		if err != nil {
			t.Errorf("mock facilitator: asset: %v", err)
			http.Error(w, "bad config", 500)
			return
		}

		authStruct := fxevm.ExactEIP3009Authorization{
			From:        auth.From,
			To:          auth.To,
			Value:       auth.Value,
			ValidAfter:  auth.ValidAfter,
			ValidBefore: auth.ValidBefore,
			Nonce:       auth.Nonce,
		}
		hashBytes, err := fxevm.HashEIP3009Authorization(authStruct, chainID, ai.Address, ai.Name, ai.Version)
		if err != nil {
			t.Errorf("mock facilitator: hash: %v", err)
			http.Error(w, "hash error", 500)
			return
		}
		hash := common.BytesToHash(hashBytes)

		sigBytes, err := hex.DecodeString(strings.TrimPrefix(sig, "0x"))
		if err != nil || len(sigBytes) != 65 {
			t.Errorf("mock facilitator: invalid signature format: %v (len=%d)", err, len(sigBytes))
			http.Error(w, "bad sig", 400)
			return
		}

		recSig := make([]byte, 65)
		copy(recSig, sigBytes)
		if recSig[64] >= 27 {
			recSig[64] -= 27
		}

		pubKey, err := crypto.SigToPub(hash.Bytes(), recSig)
		if err != nil {
			t.Errorf("mock facilitator: SigToPub failed: %v", err)
			http.Error(w, "sig recovery failed", 400)
			return
		}

		recoveredAddr := crypto.PubkeyToAddress(*pubKey)
		if recoveredAddr != expectedAddr {
			t.Errorf("mock facilitator: recovered address %s != expected %s", recoveredAddr.Hex(), expectedAddr.Hex())
			http.Error(w, "wrong signer", 400)
			return
		}

		t.Logf("mock facilitator: signer verified: %s", recoveredAddr.Hex())

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"success":true,"transaction":"0xMOCK%s","network":"base-sepolia"}`, hex.EncodeToString(nonceBytes[:8]))
	}))
}

func TestPayWithMockFacilitator(t *testing.T) {
	signer := newTestSigner(t)
	expectedValue, _ := USDCToUnits("13.42")

	merchant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Payment") == "" {
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"orderId":"PZA-001","status":"confirmed"}`)
	}))
	defer merchant.Close()

	facilitator := strengthenedMockFacilitator(t, signer.Address(), expectedValue)
	defer facilitator.Close()

	client := NewNativeClient()
	client.httpClient = &http.Client{Timeout: 10 * time.Second}

	receipt, err := client.Pay(context.Background(), PayInput{
		URL:    merchant.URL + "/orders",
		Method: http.MethodPost,
		Body:   []byte(`{"dish":"carbonara","servings":2}`),
		PaymentRequirements: PaymentRequirements{
			Scheme:            "exact",
			Network:           "base-sepolia",
			MaxAmountRequired: "13.42",
			PayTo:             common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
		Signer:         signer,
		FacilitatorURL: facilitator.URL,
	})
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if !strings.HasPrefix(receipt.TxHash, "0xMOCK") {
		t.Errorf("unexpected txHash: %s", receipt.TxHash)
	}
	if !bytes.Contains(receipt.MerchantResponse, []byte("PZA-001")) {
		t.Errorf("unexpected merchant response: %s", receipt.MerchantResponse)
	}
	t.Logf("Pay succeeded: txHash=%s nonce=%s", receipt.TxHash, receipt.NonceHex)
}

func TestDryRunNeverCallsFacilitator(t *testing.T) {
	signer := newTestSigner(t)

	facilitator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("dry-run should not call facilitator, but got %s %s", r.Method, r.URL.Path)
		http.Error(w, "should not be called", 500)
	}))
	defer facilitator.Close()

	client := NewNativeClient()
	receipt, err := client.Pay(context.Background(), PayInput{
		URL:    "http://merchant.local/orders",
		Method: http.MethodPost,
		Body:   []byte(`{}`),
		PaymentRequirements: PaymentRequirements{
			Network:           "base-sepolia",
			Scheme:            "exact",
			MaxAmountRequired: "5.00",
			PayTo:             common.HexToAddress("0x1234567890123456789012345678901234567890"),
		},
		Signer:         signer,
		DryRun:         true,
		FacilitatorURL: facilitator.URL,
	})
	if err != nil {
		t.Fatalf("dry-run Pay: %v", err)
	}
	if !strings.HasPrefix(receipt.TxHash, "0xDEADBEEF") {
		t.Errorf("expected fake txHash, got: %s", receipt.TxHash)
	}
	t.Logf("Dry-run returned: %s", receipt.TxHash)
}
