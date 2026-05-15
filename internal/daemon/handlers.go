package daemon

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jiriki-labs/agentic-wallet/internal/policy"
	"github.com/jiriki-labs/agentic-wallet/internal/x402"
)

type payX402Request struct {
	URL                 string                   `json:"url"`
	Method              string                   `json:"method"`
	Body                json.RawMessage          `json:"body,omitempty"` // optional JSON body replayed to the merchant (e.g. POST /orders)
	Merchant            string                   `json:"merchant"`
	Amount              string                   `json:"amount"`
	Token               string                   `json:"token"`
	Chain               string                   `json:"chain"`
	Reason              string                   `json:"reason"`
	Agent               string                   `json:"agent"`
	PaymentRequirements x402.PaymentRequirements `json:"paymentRequirements"`
}

type payX402Response struct {
	Status         string `json:"status"`
	PolicyDecision string `json:"policyDecision"`
	Amount         string `json:"amount,omitempty"`
	Token          string `json:"token,omitempty"`
	Chain          string `json:"chain,omitempty"`
	TxHash         string `json:"txHash,omitempty"`
	OrderID        string `json:"orderId,omitempty"`
	ApprovalID     string `json:"approvalId,omitempty"`
	ExpiresAt      string `json:"expiresAt,omitempty"`
	Error          string `json:"error,omitempty"`
}

// generateID returns a random 128-bit hex identifier for approval IDs.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) handlePayX402(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idemKey := r.Header.Get("Idempotency-Key")

	var req payX402Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}

	if idemKey != "" {
		reqBytes, _ := json.Marshal(req)
		bodyHash := sha256.Sum256(reqBytes)
		cacheKey := idemKey + ":" + hex.EncodeToString(bodyHash[:])
		if cached, ok := s.idemCache.get(cacheKey); ok {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Idempotent-Replayed", "true")
			w.WriteHeader(cached.status)
			_, _ = w.Write(cached.body)
			return
		}
		status, body := s.processPayX402(r.Context(), req)
		s.idemCache.set(cacheKey, status, body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
		return
	}

	status, body := s.processPayX402(r.Context(), req)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (s *Server) processPayX402(ctx context.Context, req payX402Request) (int, []byte) {
	policyReq := policy.PaymentRequest{
		Agent:    req.Agent,
		Merchant: req.Merchant,
		Amount:   req.Amount,
		Token:    req.Token,
		Chain:    req.Chain,
		Reason:   req.Reason,
	}

	result, err := s.policyEngine.Decide(policyReq)
	if err != nil {
		body, _ := json.Marshal(payX402Response{Status: "error", Error: err.Error()})
		return http.StatusInternalServerError, body
	}

	switch result.Decision {
	case policy.DecisionReject:
		body, _ := json.Marshal(payX402Response{
			Status:         "rejected",
			PolicyDecision: "reject",
			Amount:         req.Amount,
			Token:          req.Token,
			Chain:          req.Chain,
			Error:          result.Reason,
		})
		return http.StatusPaymentRequired, body

	case policy.DecisionDryRun:
		payInput := x402.PayInput{
			URL:                 req.URL,
			Method:              req.Method,
			Body:                []byte(req.Body),
			PaymentRequirements: req.PaymentRequirements,
			Signer:              s.signer,
			DryRun:              true,
			FacilitatorURL:      s.facilitatorURL,
		}
		receipt, err := s.x402Client.Pay(ctx, payInput)
		if err != nil {
			body, _ := json.Marshal(payX402Response{Status: "error", Error: err.Error()})
			return http.StatusInternalServerError, body
		}
		orderID := parseMerchantOrderID(receipt.MerchantResponse)
		_ = s.auditStore.UpdateTxHash(result.RecordID, receipt.TxHash, "dry-run", orderID)
		body, _ := json.Marshal(payX402Response{
			Status:         "dry-run",
			PolicyDecision: "dry-run",
			Amount:         req.Amount,
			Token:          req.Token,
			Chain:          req.Chain,
			TxHash:         receipt.TxHash,
			OrderID:        orderID,
		})
		return http.StatusOK, body

	case policy.DecisionConfirmRequired:
		approvalID := generateID()
		expiresAt := time.Now().Add(10 * time.Minute)
		payInput := x402.PayInput{
			URL:                 req.URL,
			Method:              req.Method,
			Body:                []byte(req.Body),
			PaymentRequirements: req.PaymentRequirements,
			Signer:              s.signer,
			FacilitatorURL:      s.facilitatorURL,
		}
		s.confirmQ.add(&pendingPayment{
			approvalID: approvalID,
			input:      payInput,
			auditID:    result.RecordID,
			expiresAt:  expiresAt,
		})
		body, _ := json.Marshal(payX402Response{
			Status:         "pending",
			PolicyDecision: "confirm-required",
			Amount:         req.Amount,
			Token:          req.Token,
			Chain:          req.Chain,
			ApprovalID:     approvalID,
			ExpiresAt:      expiresAt.Format(time.RFC3339),
		})
		return http.StatusAccepted, body

	case policy.DecisionApprove:
		payInput := x402.PayInput{
			URL:                 req.URL,
			Method:              req.Method,
			Body:                []byte(req.Body),
			PaymentRequirements: req.PaymentRequirements,
			Signer:              s.signer,
			FacilitatorURL:      s.facilitatorURL,
		}
		receipt, err := s.x402Client.Pay(ctx, payInput)
		if err != nil {
			_ = s.auditStore.UpdateTxHash(result.RecordID, "", "failed", "")
			body, _ := json.Marshal(payX402Response{Status: "error", Error: fmt.Sprintf("payment failed: %v", err)})
			return http.StatusBadGateway, body
		}
		orderID := parseMerchantOrderID(receipt.MerchantResponse)
		_ = s.auditStore.UpdateTxHash(result.RecordID, receipt.TxHash, "paid", orderID)
		body, _ := json.Marshal(payX402Response{
			Status:         "paid",
			PolicyDecision: "approve",
			Amount:         req.Amount,
			Token:          req.Token,
			Chain:          req.Chain,
			TxHash:         receipt.TxHash,
			OrderID:        orderID,
		})
		return http.StatusOK, body
	}

	body, _ := json.Marshal(payX402Response{Status: "error", Error: "unknown policy decision"})
	return http.StatusInternalServerError, body
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ApprovalID string `json:"approvalId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	pending, err := s.confirmQ.get(req.ApprovalID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	receipt, err := s.x402Client.Pay(r.Context(), pending.input)
	if err != nil {
		s.confirmQ.remove(req.ApprovalID)
		_ = s.auditStore.UpdateTxHash(pending.auditID, "", "failed", "")
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("payment failed: %v", err)})
		return
	}

	s.confirmQ.remove(req.ApprovalID)
	orderID := parseMerchantOrderID(receipt.MerchantResponse)
	_ = s.auditStore.UpdateTxHash(pending.auditID, receipt.TxHash, "paid", orderID)
	writeJSON(w, http.StatusOK, payX402Response{
		Status:         "paid",
		PolicyDecision: "approve",
		TxHash:         receipt.TxHash,
		OrderID:        orderID,
	})
}
