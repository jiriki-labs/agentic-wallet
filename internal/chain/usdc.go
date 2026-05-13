package chain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// balanceOfSelector is the first 4 bytes of keccak256("balanceOf(address)")
var balanceOfSelector = crypto.Keccak256([]byte("balanceOf(address)"))[:4]

// USDCBalance returns the USDC balance for an address (6 decimal places).
func (c *Client) USDCBalance(ctx context.Context, usdcAddr, walletAddr common.Address) (*big.Int, error) {
	// Encode balanceOf(address) call
	data := make([]byte, 4+32)
	copy(data[:4], balanceOfSelector)
	copy(data[4+12:], walletAddr.Bytes())

	msg := ethereum.CallMsg{
		To:   &usdcAddr,
		Data: data,
	}
	result, err := c.ec.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("balanceOf call: %w", err)
	}
	if len(result) < 32 {
		return nil, fmt.Errorf("unexpected result length: %d", len(result))
	}
	balance := new(big.Int).SetBytes(result[:32])
	return balance, nil
}

// FormatUSDC formats a USDC balance (6 decimals) as a human-readable string.
func FormatUSDC(units *big.Int) string {
	whole := new(big.Int).Div(units, big.NewInt(1_000_000))
	remainder := new(big.Int).Mod(units, big.NewInt(1_000_000))
	return fmt.Sprintf("%s.%06d", whole.String(), remainder.Int64())
}
