package chain

import (
	"math/big"
	"testing"
)

func TestFormatETH(t *testing.T) {
	t.Parallel()
	tests := []struct {
		wei  string
		want string
	}{
		{"0", "0"},
		{"1000000000000000000", "1"},
		{"1500000000000000000", "1.5"},
		{"1", "0.000000000000000001"},
	}
	for _, tc := range tests {
		wei, ok := new(big.Int).SetString(tc.wei, 10)
		if !ok {
			t.Fatalf("bad wei %q", tc.wei)
		}
		if got := FormatETH(wei); got != tc.want {
			t.Errorf("FormatETH(%s) = %q, want %q", tc.wei, got, tc.want)
		}
	}
}

func TestFormatUSDC(t *testing.T) {
	t.Parallel()
	units := big.NewInt(8_500_000)
	if got := FormatUSDC(units); got != "8.500000" {
		t.Errorf("FormatUSDC = %q, want 8.500000", got)
	}
}
