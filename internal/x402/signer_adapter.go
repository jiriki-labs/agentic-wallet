package x402

import (
	"context"

	fxevm "github.com/x402-foundation/x402/go/mechanisms/evm"
)

// evmSignerAdapter bridges Jiriki's hash-only keystore signer to x402-foundation's
// EIP-712 typed-data signing surface.
type evmSignerAdapter struct {
	s Signer
}

func (a *evmSignerAdapter) Address() string {
	return a.s.Address().Hex()
}

func (a *evmSignerAdapter) SignTypedData(
	_ context.Context,
	domain fxevm.TypedDataDomain,
	types map[string][]fxevm.TypedDataField,
	primaryType string,
	message map[string]interface{},
) ([]byte, error) {
	digest, err := fxevm.HashTypedData(domain, types, primaryType, message)
	if err != nil {
		return nil, err
	}
	sig, err := a.s.SignHash(digest)
	if err != nil {
		return nil, err
	}
	// crypto.Sign returns v as 0 or 1; EIP-3009 / ecrecover convention requires 27 or 28.
	if len(sig) == 65 && sig[64] < 27 {
		sig[64] += 27
	}
	return sig, nil
}
