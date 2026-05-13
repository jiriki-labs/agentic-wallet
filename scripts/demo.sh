#!/usr/bin/env bash
# Grocery402 demo: recipe fetch + optional pay-x402 via Jiriki (Unix socket).
# Prerequisites:
#   - Grocery API on http://localhost:4402 (e.g. `make grocery-dev` from another terminal)
#   - `jiriki up` with policy allowing merchant localhost:4402 (see configs/policy.example.yaml)
# Environment:
#   MERCHANT_ADDR — must match paymentRequirements.payTo (default: Anvil-style test address)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

MERCHANT_ADDR="${MERCHANT_ADDR:-0x1234567890123456789012345678901234567890}"
JIRIKI_HOME="${JIRIKI_HOME:-$HOME/.config/jiriki}"
SOCK="${JIRIKI_HOME}/jiriki.sock"
GROCERY_URL="${GROCERY_URL:-http://localhost:4402}"

echo "=== Jiriki Grocery402 demo ==="
echo "Grocery API: ${GROCERY_URL}"

if ! curl -sf "${GROCERY_URL}/recipes?dish=carbonara&servings=2" >/dev/null; then
  echo "ERROR: Grocery API not reachable at ${GROCERY_URL}" >&2
  echo "Start it with: make grocery-dev   (from repo root)" >&2
  exit 1
fi

echo ""
echo "User: I'd like carbonara."
echo ""
echo "Agent: fetching recipe..."
curl -sS "${GROCERY_URL}/recipes?dish=carbonara&servings=2" | {
  command -v jq >/dev/null && jq . || cat
}

if [[ ! -S "$SOCK" ]]; then
  echo ""
  echo "No Unix socket at $SOCK — skipping pay-x402 (start jiriki up in another terminal)."
  exit 0
fi

echo ""
echo "Agent: POST /orders (unpaid) to read payment requirements..."
PR_FILE="$(mktemp -u /tmp/g402demoXXXXXX)"
HTTP_CODE="$(curl -sS -o "$PR_FILE" -w '%{http_code}' -X POST "${GROCERY_URL}/orders" \
  -H 'Content-Type: application/json' \
  -d '{"dish":"carbonara","servings":2}' || true)"
if [[ "$HTTP_CODE" != "402" ]]; then
  echo "Expected HTTP 402 from /orders, got ${HTTP_CODE}. Body:" >&2
  cat "$PR_FILE" >&2
  rm -f "$PR_FILE"
  exit 1
fi

PAY_JSON="$(
  GROCERY_URL="$GROCERY_URL" python3 - "$PR_FILE" <<'PY'
import json, os, sys
path = sys.argv[1]
with open(path, encoding="utf-8") as f:
    pr = json.load(f)
req = pr.get("paymentRequirements") or {}
base = os.environ["GROCERY_URL"].rstrip("/")
body = {
    "url": base + "/orders",
    "method": "POST",
    "body": {"dish": "carbonara", "servings": 2},
    "merchant": "localhost:4402",
    "amount": req.get("maxAmountRequired", "8.50"),
    "token": "USDC",
    "chain": "base-sepolia",
    "reason": "ingredients for carbonara",
    "paymentRequirements": req,
}
print(json.dumps(body))
PY
)"
rm -f "$PR_FILE"

IDEM="$(python3 -c 'import uuid; print(uuid.uuid4())')"
echo ""
echo "Agent: pay-x402 via Jiriki (Idempotency-Key=${IDEM})..."
curl -sS --unix-socket "$SOCK" \
  -X POST http://localhost/pay-x402 \
  -H 'Content-Type: application/json' \
  -H "Idempotency-Key: ${IDEM}" \
  -d "$PAY_JSON" | {
  command -v jq >/dev/null && jq . || cat
}

echo ""
echo "Done."
