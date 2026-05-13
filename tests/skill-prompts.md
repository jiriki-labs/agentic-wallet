# Skill replay fixtures — Grocery402 / `pay-x402`

This file captures **five user-style intents** and the **daemon request shape** an agent should synthesize after resolving the dish and fetching `paymentRequirements` from `POST /orders` (HTTP 402).

Policy outcomes depend on the user’s live `policy.yaml` (mode, limits). These examples focus on **valid JSON** accepted by `POST /pay-x402`.

---

## 1) „I need some carbonara”

**Resolved dish:** `carbonara`, servings `2`  
**Recipe total (API):** `8.50` USDC  
**Idempotency-Key (example):** `a1111111-1111-4111-8111-111111111111`

**`POST /pay-x402` body (shape):**

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
  "paymentRequirements": "<paste from POST /orders 402 JSON>"
}
```

**Example daemon responses (depends on policy):**

- `mode: confirm` → `202`, `"status":"pending"`, `"approvalId":"…"`
- `mode: auto` + funded wallet → `200`, `"status":"paid"`, `"txHash":"0x…"`
- `mode: dry-run` → `200`, `"status":"dry-run"`

---

## 2) „I'd like some pasta”

**Agent resolution:** map to **`carbonara`** (or ask one clarifying question first).  
**Servings:** `2`  
**Idempotency-Key (example):** `b2222222-2222-4222-8222-222222222222`

**`POST /pay-x402` body:** same structure as (1) with `reason` e.g. `"ingredients for carbonara"` and matching `body` / `amount` / `paymentRequirements`.

---

## 3) „zrób mi zakupy na makaron”

**Agent resolution:** **`bolognese`** (pasta-focused grocery).  
**Servings:** `2`  
**Idempotency-Key (example):** `c3333333-3333-4333-8333-333333333333`

**Notes:** Call `GET /recipes?dish=bolognese&servings=2` for the live **`totalUsdc`**, then `POST /orders` for **`paymentRequirements`**.

---

## 4) „hungry, something Italian”

**Agent resolution:** propose **`carbonara`** or **`aglio e olio`**; if choosing **aglio e olio**, use that dish in `body` and amounts from the recipe endpoint.  
**Idempotency-Key (example):** `d4444444-4444-4444-8444-444444444444`

---

## 5) „carbonara ingredients please”

**Resolved dish:** `carbonara`, servings `2`  
**Idempotency-Key (example):** `e5555555-5555-4555-8555-555555555555`

**`POST /pay-x402` body:** identical pattern to (1); always refresh **`paymentRequirements`** from a fresh `402` from the merchant before paying.

---

## Acceptance notes (from MVP spec)

- All five prompts should yield a **`pay-x402`** JSON object that passes daemon decoding (required string fields + `paymentRequirements` matching merchant 402).
- At least **three** prompts may reach a **`policyDecision`** in `auto` / `dry-run` / `confirm` paths; others may **`reject`** on policy without invalidating the exercise.
