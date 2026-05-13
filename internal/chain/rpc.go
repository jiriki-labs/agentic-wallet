package chain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const BaseSepoliaRPC = "https://sepolia.base.org"

// Client wraps ethclient for balance queries.
type Client struct {
	ec *ethclient.Client
}

// New creates a new chain client.
func New(rpcURL string) (*Client, error) {
	ec, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", rpcURL, err)
	}
	return &Client{ec: ec}, nil
}

// Close closes the connection.
func (c *Client) Close() {
	c.ec.Close()
}

// ETHBalance returns the ETH balance for an address in wei.
func (c *Client) ETHBalance(ctx context.Context, addr common.Address) (*big.Int, error) {
	return c.ec.BalanceAt(ctx, addr, nil)
}
