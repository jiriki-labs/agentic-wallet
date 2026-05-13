package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "The Grocery402 merchant demo is implemented in TypeScript (Nest + Next.js).")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  cd apps/grocery && npm install && npm run dev")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "API defaults to http://127.0.0.1:4402 (set PORT). Web UI: http://127.0.0.1:3020")
	fmt.Fprintln(os.Stderr, "Set MERCHANT_ADDR (0x…) for x402 payTo. GROCERY_SKIP_X402_VERIFY=1 skips facilitator verify (local only).")
	os.Exit(0)
}
