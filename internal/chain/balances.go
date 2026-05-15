package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// WalletBalances holds on-chain ETH (wei) and USDC (6-decimal units) balances.
type WalletBalances struct {
	ETH  *big.Int
	USDC *big.Int
}

// WalletBalances fetches native ETH and ERC-20 USDC balances for wallet.
func (c *Client) WalletBalances(ctx context.Context, usdcContract, wallet common.Address) (WalletBalances, error) {
	eth, err := c.ETHBalance(ctx, wallet)
	if err != nil {
		return WalletBalances{}, fmt.Errorf("eth balance: %w", err)
	}
	usdc, err := c.USDCBalance(ctx, usdcContract, wallet)
	if err != nil {
		return WalletBalances{}, fmt.Errorf("usdc balance: %w", err)
	}
	return WalletBalances{ETH: eth, USDC: usdc}, nil
}

// FormatETH formats a wei balance (18 decimals) as a decimal string.
func FormatETH(wei *big.Int) string {
	if wei == nil {
		return "0"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	whole := new(big.Int).Div(wei, divisor)
	rem := new(big.Int).Mod(wei, divisor)
	frac := rem.String()
	if len(frac) < 18 {
		frac = strings.Repeat("0", 18-len(frac)) + frac
	}
	frac = strings.TrimRight(frac, "0")
	if frac == "" {
		return whole.String()
	}
	return whole.String() + "." + frac
}
