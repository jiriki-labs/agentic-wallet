package x402

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	x402f "github.com/x402-foundation/x402/go"
	x402http "github.com/x402-foundation/x402/go/http"
	exactv1 "github.com/x402-foundation/x402/go/mechanisms/evm/exact/v1/client"
	fxtypes "github.com/x402-foundation/x402/go/types"
)

const defaultFacilitatorURL = "https://x402.org/facilitator"

// DefaultUSDCAddress is canonical USDC on Base Sepolia (demo / policy examples).
const DefaultUSDCAddress = "0x036CbD53842c5426634e7929541eC2318f3dCF7e"

// Client is the x402 payment client interface.
type Client interface {
	Pay(ctx context.Context, input PayInput) (Receipt, error)
}

// NativeClient implements Client using github.com/x402-foundation/x402/go (V1 exact + HTTP facilitator).
type NativeClient struct {
	httpClient *http.Client
	// defaultFacilitator is applied when PayInput.FacilitatorURL is empty (daemon --facilitator-url).
	defaultFacilitator string
}

// SetFacilitatorURL sets the facilitator base URL used when PayInput.FacilitatorURL is empty.
func (c *NativeClient) SetFacilitatorURL(url string) {
	c.defaultFacilitator = strings.TrimSuffix(strings.TrimSpace(url), "/")
}

// NewNativeClient returns a client configured for Base Sepolia USDC flows.
func NewNativeClient() *NativeClient {
	return &NativeClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Pay builds a V1 payment payload (EIP-3009), settles via the facilitator, then replays the merchant request.
func (c *NativeClient) Pay(ctx context.Context, input PayInput) (Receipt, error) {
	if input.DryRun {
		return c.dryRunPay(ctx, input)
	}

	if input.PaymentRequirements.MaxTimeoutSeconds > 0 {
		validAfter := time.Now().Unix() - 5
		validBefore := time.Now().Unix() + int64(input.PaymentRequirements.MaxTimeoutSeconds)
		if err := ValidateWindow(validAfter, validBefore); err != nil {
			return Receipt{}, fmt.Errorf("payment requirements validation: %w", err)
		}
	}

	atomicAmount, err := decimalUSDCToAtomicString(input.PaymentRequirements.MaxAmountRequired)
	if err != nil {
		return Receipt{}, err
	}

	reqV1 := toPaymentRequirementsV1(input.PaymentRequirements, atomicAmount)
	net := x402f.Network(strings.TrimSpace(input.PaymentRequirements.Network))
	if net == "" {
		net = "base-sepolia"
	}

	x402Client := x402f.Newx402Client().RegisterV1(net, exactv1.NewExactEvmSchemeV1(&evmSignerAdapter{s: input.Signer}))
	payload, err := x402Client.CreatePaymentPayloadV1(ctx, reqV1)
	if err != nil {
		return Receipt{}, fmt.Errorf("create payment payload: %w", err)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Receipt{}, err
	}

	reqBytes, err := json.Marshal(reqV1)
	if err != nil {
		return Receipt{}, err
	}

	facilitatorURL := strings.TrimSuffix(strings.TrimSpace(input.FacilitatorURL), "/")
	if facilitatorURL == "" {
		facilitatorURL = c.defaultFacilitator
	}
	if facilitatorURL == "" {
		facilitatorURL = defaultFacilitatorURL
	}

	fac := x402http.NewHTTPFacilitatorClient(&x402http.FacilitatorConfig{
		URL:        facilitatorURL,
		HTTPClient: c.httpClient,
	})
	settleResp, err := fac.Settle(ctx, payloadBytes, reqBytes)
	if err != nil {
		return Receipt{}, fmt.Errorf("facilitator settle: %w", err)
	}
	if !settleResp.Success {
		return Receipt{}, fmt.Errorf("facilitator rejected payment: reason=%s message=%s", settleResp.ErrorReason, settleResp.ErrorMessage)
	}

	proofEncoded := base64.StdEncoding.EncodeToString(payloadBytes)
	merchantResp, err := c.replayRequest(ctx, input, proofEncoded)
	if err != nil {
		return Receipt{}, fmt.Errorf("replay request: %w", err)
	}

	nonceHex, err := nonceHexFromPayloadV1(payload)
	if err != nil {
		return Receipt{}, err
	}

	return Receipt{
		TxHash:           settleResp.Transaction,
		Proof:            proofEncoded,
		MerchantResponse: merchantResp,
		NonceHex:         nonceHex,
	}, nil
}

func (c *NativeClient) dryRunPay(ctx context.Context, input PayInput) (Receipt, error) {
	atomicAmount, err := decimalUSDCToAtomicString(input.PaymentRequirements.MaxAmountRequired)
	if err != nil {
		return Receipt{}, err
	}
	reqV1 := toPaymentRequirementsV1(input.PaymentRequirements, atomicAmount)
	net := x402f.Network(strings.TrimSpace(input.PaymentRequirements.Network))
	if net == "" {
		net = "base-sepolia"
	}
	x402Client := x402f.Newx402Client().RegisterV1(net, exactv1.NewExactEvmSchemeV1(&evmSignerAdapter{s: input.Signer}))
	payload, err := x402Client.CreatePaymentPayloadV1(ctx, reqV1)
	if err != nil {
		return Receipt{}, err
	}
	nonceHex, err := nonceHexFromPayloadV1(payload)
	if err != nil {
		return Receipt{}, err
	}
	if len(nonceHex) < 8 {
		return Receipt{}, fmt.Errorf("unexpected nonce length")
	}
	fakeTxHash := "0xDEADBEEF000000000000000000000000000000000000000000000000" + nonceHex[:8]
	return Receipt{
		TxHash:   fakeTxHash,
		Proof:    "dry-run",
		NonceHex: nonceHex,
	}, nil
}

func toPaymentRequirementsV1(pr PaymentRequirements, maxAmountAtomic string) fxtypes.PaymentRequirementsV1 {
	asset := strings.TrimSpace(pr.Asset)
	if asset == "" {
		asset = DefaultUSDCAddress
	}
	var extra *json.RawMessage
	if pr.Extra != nil {
		if raw, err := json.Marshal(pr.Extra); err == nil {
			e := json.RawMessage(raw)
			extra = &e
		}
	}
	return fxtypes.PaymentRequirementsV1{
		Scheme:            pr.Scheme,
		Network:           pr.Network,
		MaxAmountRequired: maxAmountAtomic,
		Resource:          pr.Resource,
		Description:       pr.Description,
		MimeType:          pr.MimeType,
		PayTo:             pr.PayTo.Hex(),
		MaxTimeoutSeconds: pr.MaxTimeoutSeconds,
		Asset:             asset,
		Extra:             extra,
	}
}

func nonceHexFromPayloadV1(p fxtypes.PaymentPayloadV1) (string, error) {
	auth, ok := p.Payload["authorization"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("payment payload: missing authorization")
	}
	raw, ok := auth["nonce"].(string)
	if !ok || raw == "" {
		return "", fmt.Errorf("payment payload: missing nonce")
	}
	return strings.TrimPrefix(strings.TrimPrefix(raw, "0x"), "0X"), nil
}

func (c *NativeClient) replayRequest(ctx context.Context, input PayInput, proof string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, input.Method, input.URL, bytes.NewReader(input.Body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Payment", proof)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
