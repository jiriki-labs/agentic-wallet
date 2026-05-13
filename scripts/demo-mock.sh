#!/usr/bin/env bash
# CI-friendly checks: Go x402 (mock facilitator) + Grocery Nest API e2e (skip on-chain verify).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "=== demo-mock: Go x402 package tests ==="
go test ./internal/x402 -count=1

echo ""
echo "=== demo-mock: Grocery API e2e (GROCERY_SKIP_X402_VERIFY) ==="
export MERCHANT_ADDR="${MERCHANT_ADDR:-0x1234567890123456789012345678901234567890}"
export GROCERY_SKIP_X402_VERIFY=1
(cd apps/grocery && npm run test:e2e -w api)

echo ""
echo "demo-mock: OK"
