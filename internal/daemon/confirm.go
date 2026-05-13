package daemon

import (
	"fmt"
	"sync"
	"time"

	"github.com/jiriki-labs/agentic-wallet/internal/x402"
)

// pendingPayment holds the prepared (but unsigned) payment material for confirm mode.
// This is in-memory only; daemon restart loses pending state (explicit MVP tradeoff).
type pendingPayment struct {
	approvalID string
	input      x402.PayInput
	auditID    int64
	expiresAt  time.Time
}

// confirmQueue manages pending confirm-mode payments.
type confirmQueue struct {
	mu      sync.Mutex
	pending map[string]*pendingPayment
}

func newConfirmQueue() *confirmQueue {
	return &confirmQueue{pending: make(map[string]*pendingPayment)}
}

func (q *confirmQueue) add(p *pendingPayment) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending[p.approvalID] = p
}

func (q *confirmQueue) get(approvalID string) (*pendingPayment, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	p, ok := q.pending[approvalID]
	if !ok {
		return nil, fmt.Errorf("no pending payment with approvalID %q", approvalID)
	}
	if time.Now().After(p.expiresAt) {
		delete(q.pending, approvalID)
		return nil, fmt.Errorf("pending payment %q has expired", approvalID)
	}
	return p, nil
}

func (q *confirmQueue) remove(approvalID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.pending, approvalID)
}
