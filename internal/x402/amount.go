package x402

import (
	"fmt"
	"math/big"
)

// USDCToUnits parses a decimal USDC amount (e.g. "8.50") into 6-decimal atomic units.
func USDCToUnits(amount string) (*big.Int, error) {
	var f float64
	if _, err := fmt.Sscanf(amount, "%f", &f); err != nil {
		return nil, fmt.Errorf("parse USDC amount %q: %w", amount, err)
	}
	return big.NewInt(int64(f * 1_000_000)), nil
}

// decimalUSDCToAtomicString returns MaxAmountRequired as a base-10 integer string
// in USDC smallest units (required by x402-foundation exact/v1).
func decimalUSDCToAtomicString(amount string) (string, error) {
	u, err := USDCToUnits(amount)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
