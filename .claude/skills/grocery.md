# Grocery402 — Order ingredients via Jiriki

## Purpose

When the user mentions food, a dish, being hungry, or grocery intent (including Polish phrases like *„zjadłbym carbonarę”*), this skill should:

1. Fetch a recipe from the **Grocery402** HTTP API (Nest.js in `apps/grocery`, default `http://localhost:4402`).
2. Show ingredients and the **total USDC** price.
3. After explicit user confirmation, request payment through the **Jiriki** wallet daemon (`POST /pay-x402`), which performs the x402 flow against `POST /orders`.

## Security (mandatory)

- **Never** log or echo the daemon **bearer token** or the contents of `~/.config/jiriki/auth`.
- **Never** paste secrets into chat or commit them to git.
- Prefer **Unix socket** transport when `~/.config/jiriki/jiriki.sock` exists.

## Step 1 — Free recipe

```bash
curl -s "http://localhost:4402/recipes?dish=DISH_NAME&servings=2"
```

Supported dishes (case-insensitive `dish` query): `carbonara`, `bolognese`, `aglio e olio`.

Present ingredients and **totalUsdc** clearly.

## Step 2 — Confirm

Ask explicitly, e.g. *“Ingredients for carbonara (2 servings) are 8.50 USDC. Place the order?”*

## Step 3 — Obtain exact `paymentRequirements`

Do **not** invent partial payment fields. The merchant returns the canonical object:

```bash
curl -sS -o /tmp/g402.json -w "%{http_code}" -X POST "http://localhost:4402/orders" \
  -H "Content-Type: application/json" \
  -d '{"dish":"DISH_NAME","servings":SERVINGS}'
```

Expect **HTTP 402**. Parse JSON and read **`paymentRequirements`** (also duplicated in header **`X-Payment-Requirements`** as raw JSON).

## Step 4 — Pay via Jiriki (`/pay-x402`)

Generate a **fresh UUID v4** per user-facing order attempt and send it as **`Idempotency-Key`**.

The daemon replays the merchant HTTP request using optional JSON field **`body`** (same object you would send to `/orders`).

```bash
IDEM_KEY=$(python3 -c "import uuid; print(uuid.uuid4())")

# Build PAY_JSON in your environment: url, method, body { dish, servings },
# merchant (host:port, e.g. localhost:4402), amount, token, chain, reason,
# paymentRequirements = exact object from step 3.

curl -sS --unix-socket "${JIRIKI_HOME:-$HOME/.config/jiriki}/jiriki.sock" \
  -X POST http://localhost/pay-x402 \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: ${IDEM_KEY}" \
  -d "${PAY_JSON}"
```

**`PAY_JSON` shape (illustrative)** — always copy **`paymentRequirements`** from the 402 response:

```json
{
  "url": "http://localhost:4402/orders",
  "method": "POST",
  "body": { "dish": "carbonara", "servings": 2 },
  "merchant": "localhost:4402",
  "amount": "8.50",
  "token": "USDC",
  "chain": "base-sepolia",
  "reason": "ingredients for carbonara",
  "paymentRequirements": { }
}
```

## Step 5 — Interpret daemon response

- **`status` = `paid`** — *“Ordered. Tx: [txHash]. Delivery ETA ~2h (demo).”*
- **`status` = `pending`** (policy confirm mode) — *“Awaiting approval. Run: `jiriki approve <approvalId>`”*
- **`status` = `rejected`** — explain **`error`**
- **`status` = `dry-run`** — policy `dry-run`: no chain settlement; tx hash is synthetic.

## Step 6 — Optional audit

```bash
curl -sS --unix-socket "${JIRIKI_HOME:-$HOME/.config/jiriki}/jiriki.sock" http://localhost/transactions | head -5
```

Or: `jiriki audit --since=1h` when implemented.

## Local stack

- API: `make grocery-dev` (from repo root) or `cd apps/grocery && npm install && npm run dev`
- Daemon: `jiriki up` with policy allowing merchant **`localhost:4402`** (see `configs/policy.example.yaml`).
