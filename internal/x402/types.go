package x402

import (
	"github.com/ethereum/go-ethereum/common"
)

// PaymentRequirements is the 402 response payload from the merchant (daemon JSON).
type PaymentRequirements struct {
	Scheme            string         `json:"scheme"`
	Network           string         `json:"network"`
	MaxAmountRequired string         `json:"maxAmountRequired"`
	Resource          string         `json:"resource"`
	Description       string         `json:"description"`
	MimeType          string         `json:"mimeType"`
	PayTo             common.Address `json:"payTo"`
	MaxTimeoutSeconds int            `json:"maxTimeoutSeconds"`
	Asset             string         `json:"asset"` // USDC contract address
	Extra             interface{}    `json:"extra,omitempty"`
}

// PayInput is the input to Client.Pay.
type PayInput struct {
	URL                 string
	Method              string
	Body                []byte
	PaymentRequirements PaymentRequirements
	Signer              Signer
	DryRun              bool
	FacilitatorURL      string // override; uses default if empty
}

// Receipt is the result of a successful payment.
type Receipt struct {
	TxHash           string
	Proof            string // base64-encoded payment proof/header
	MerchantResponse []byte
	NonceHex         string
}

// Signer can sign a 32-byte hash (EIP-712 digest).
type Signer interface {
	Address() common.Address
	SignHash(hash []byte) ([]byte, error)
}
