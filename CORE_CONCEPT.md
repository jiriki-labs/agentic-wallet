## 1. Working title

**Jiriki**
Demo: **Grocery402**

## 2. One-sentence description

Jiriki is a local, lightweight wallet for AI agents that lets them safely make payments on the user's behalf within clearly defined limits and policies.

The Grocery402 demo illustrates a simple scenario: the user mentions a dish or groceries, and the agent independently goes through the recipe, payment for ingredients, and order confirmation.

## 3. The problem

AI agents increasingly handle operational tasks: searching for information, calling APIs, comparing options, making decisions, and invoking tools. They still struggle to complete a full economic transaction.

Today a typical agent can:

* find a product or service,
* fill out a form,
* prepare a recommendation,
* generate a payment link.

But it usually cannot safely:

* pay for an API resource,
* pay for a service,
* fulfill orders,
* act autonomously within the user's budget.

There is no layer that gives the agent the ability to pay without handing it full control over money.

## 4. Why now

Three parallel trends are converging:

1. **AI agents are moving beyond chat**
   More agents use tools, APIs, and external services.
2. **Programmable payments are maturing**
   Stablecoins, EVM wallets, ENS, and protocols like x402 let you treat payment as part of request–response, not a separate checkout.
3. **HTTP 402 may finally get practical use**
   An API can respond: "this resource costs X"; the agent can pay and continue the request.

## 5. Who it is for

### Primary audience

Teams building AI agents that want to let agents perform paid actions in a controlled way.

### Example users

* AI agent builders,
* developers integrating with Claude / ChatGPT / local agents,
* companies building tools that automate shopping, research, paid API access, and on-demand services,
* paid API creators who want to sell access to agents without classic user onboarding.

## 6. Core product thesis

The agent should not have full control over the user's wallet.

The agent should have access to a **controlled ability to pay**:

* only to allowed recipients,
* only in a specified token,
* only up to a specified amount,
* only in a specified context,
* with a full log and auditability.

## 7. Product scope

### What we are building

We are building a local wallet daemon that runs on the agent's machine and exposes a controlled API for executing payments.

The agent does not receive the private key. The agent can only submit a payment request.

The wallet daemon:

* holds a local encrypted key,
* checks security policies,
* signs the payment,
* executes the payment via x402 or a compatible payment flow,
* writes an audit log,
* returns the result to the agent.

### What we are not building initially

We are not building a full consumer wallet like MetaMask.

Initially we are not building:

* a browser extension,
* a mobile wallet,
* our own chain,
* our own stablecoin,
* a full marketplace,
* a complex account abstraction system,
* a recovery flow for mass-market users.

## 8. Demo: Grocery402

Grocery402 is a demo showing that an agent can fetch a recipe, order ingredients, and pay for them within the user's policy.

### Demo scenario

The user writes to the agent:

> I'd like carbonara — order the ingredients, up to about 15 USDC.

The agent:

1. Recognizes the intent (dish / groceries).
2. Fetches the recipe and prices from the demo API (`GET /recipes`).
3. Validates ingredients and total cost in USDC.
4. Learns that the order requires payment (HTTP 402 / x402).
5. Asks the local wallet daemon to execute the payment.
6. The wallet daemon checks policy.
7. If the payment fits within limits, it signs and sends the transaction.
8. The agent retries the API request with proof of payment.
9. The merchant API returns confirmation of the ingredient order.
10. The agent reports the outcome to the user.

### Example agent reply

> I ordered ingredients for carbonara (pancetta, eggs, pecorino, pasta, pepper). Total cost: 8.50 USDC. The order was within the limit. Estimated delivery: ~2 h. Order ID: GRC-001.

## 9. How it works technically

### High-level architecture

```txt
[User]
  |
  v
[Agent / Claude Skill]
  |
  v
[Grocery ordering tool]
  |
  v
[Local Wallet Daemon]
  |
  +-- Policy Engine
  +-- Encrypted Local Keystore
  +-- x402 Payment Client
  +-- Audit Log
  |
  v
[Grocery402 demo merchant API]
```

### Payment flow

```txt
1. Agent sends a request to the merchant API (e.g. ingredient order).
2. Merchant API responds: 402 Payment Required.
3. Agent forwards payment requirements to the wallet daemon.
4. Wallet daemon checks policy.
5. Wallet daemon signs and sends the payment.
6. Agent retries the request with proof of payment.
7. Merchant API confirms the order.
```

## 10. System components

### 10.1 Agent skill / tool

The layer used by the AI agent.

It is responsible for:

* interpreting the user's intent,
* choosing the right tool,
* preparing requests to the merchant API (recipes, orders),
* handling `402 Payment Required` responses,
* talking to the wallet daemon,
* reporting results to the user.

The agent does not know the private key and has no direct access to funds.

### 10.2 Wallet daemon

A local process running on the user's machine or the agent machine.

It is responsible for:

* storing the encrypted key,
* unlocking the wallet,
* signing payments,
* enforcing policies,
* talking to the chain / facilitator,
* maintaining the audit log.

Preferred implementation: **Go** as a single lightweight binary.

### 10.3 Policy engine

The most important security component.

Example policies:

```yaml
allowedTokens:
  - USDC

allowedChains:
  - base-sepolia
  - base

allowedMerchants:
  - localhost:4402
  - grocery-demo.local

maxAmountPerRequest: "15.00"
dailyLimit: "50.00"
requireApprovalAbove: "20.00"
mode: "confirm"
```

The policy engine decides whether a given payment can run automatically, requires confirmation, or must be blocked.

### 10.4 Local keystore

Local encrypted storage for the private key.

Assumptions:

* the private key never reaches the model,
* the private key is not stored in `.env`,
* the key can be unlocked with a password at startup,
* eventually: integration with OS keychain, Secure Enclave, TPM, or a hardware wallet.

### 10.5 Audit log

Every payment attempt should be recorded.

Example entry:

```json
{
  "timestamp": "2026-05-11T12:00:00Z",
  "agent": "claude-skill-demo",
  "merchant": "localhost:4402",
  "amount": "8.50",
  "token": "USDC",
  "chain": "base-sepolia",
  "reason": "grocery ingredients for carbonara",
  "policyDecision": "approved",
  "txHash": "0x...",
  "orderId": "GRC-001"
}
```

## 11. Operating modes

### Dry run

The agent simulates payment but signs nothing.

Use for:

* development,
* tests,
* risk-free demos,
* flow debugging.

### Confirm mode

The wallet daemon requires local approval before executing a payment.

Use for:

* first real demos,
* tests with real stablecoin,
* higher-risk scenarios.

### Auto-limited mode

The wallet daemon executes payments automatically, but only within configured limits.

Use for:

* target agentic flows,
* low-amount payments,
* paid API access,
* recurring automations.

## 12. Wallet daemon API

### `GET /balance`

Returns available balance.

### `GET /policy`

Returns the active payment policy.

### `POST /pay-x402`

Executes a payment according to x402 payment requirements.

Example body:

```json
{
  "url": "http://localhost:4402/orders",
  "method": "POST",
  "merchant": "localhost:4402",
  "amount": "8.50",
  "token": "USDC",
  "chain": "base-sepolia",
  "reason": "ingredients order requested by user",
  "paymentRequirements": {}
}
```

Example response:

```json
{
  "status": "paid",
  "policyDecision": "approved",
  "amount": "8.50",
  "token": "USDC",
  "chain": "base-sepolia",
  "txHash": "0x..."
}
```

### `GET /transactions`

Returns history of payment attempts.

### `POST /approve`

Used to manually approve payments in confirm mode.

## 13. Security

### Core principles

1. The agent never receives the private key.
2. The wallet daemon runs locally.
3. The wallet daemon accepts requests only from localhost or a Unix socket.
4. Every payment goes through the policy engine.
5. Every payment attempt is logged.
6. The default mode is dry-run or confirm, not auto.
7. In production, use a separate wallet with a small budget.

### Potential risks

* prompt injection may push the agent to attempt a payment,
* a malicious API may try to force a payment,
* local malware may try to talk to the daemon,
* misconfigured policy may allow overspending,
* the agent may misinterpret the user's intent.

### Mitigations

* merchant allowlists,
* amount limits,
* local auth token,
* Unix socket instead of a public HTTP port,
* human confirmation for larger amounts,
* dry-run mode for tests,
* a separate wallet with low balance,
* full audit log.

## 14. MVP

### MVP goal

Prove that an agent can perform a paid action on the user's behalf without access to the private key and with policy control.

### MVP scope

The MVP should include:

1. Go wallet daemon.
2. Local encrypted keystore.
3. Simple policy engine.
4. Audit log in SQLite.
5. Integration with x402 payment flow.
6. Grocery402 demo merchant API (recipes + ingredient orders).
7. Agent tool / Claude skill for the grocery scenario (Grocery402).
8. Dry-run and confirm modes.
9. Simple CLI or local panel showing payments.

### Out of MVP scope

* production custody,
* mobile app,
* full multi-chain support,
* real integrations with wholesalers / food suppliers,
* heavy UI,
* wallet recovery,
* multi-user SaaS.

## 15. User stories

### User

As a user I want to tell the agent to order ingredients for a chosen dish so I do not have to go through checkout manually.

### User

As a user I want to set a maximum budget so the agent cannot spend more than I allow.

### User

As a user I want to see payment history so I know what the agent did with my funds.

### Agent developer

As a developer I want a local wallet API so the agent can request payment without access to the private key.

### Merchant / API provider

As a merchant I want to return `402 Payment Required` so the agent can pay automatically and continue the request.

## 16. Demo success

The demo succeeds if you can show the full flow:

1. The user gives a natural instruction.
2. The agent places the order.
3. The merchant requires payment.
4. The wallet daemon checks policy.
5. Payment is approved.
6. The order is confirmed.
7. The audit log shows what happened.

## 17. Key message

We are not building a wallet for people.

We are building a wallet for agents that people can control.

The product is not that the agent has money. The product is that the agent has a limited, auditable, safe ability to execute payments.

## 18. Short pitch statement

**Jiriki lets AI agents safely pay for digital and real-world services using programmable money, while users stay in control through local policies, limits, and audit logs.**

## 19. Demo pitch

**Grocery402 shows the future of agentic payments: the user mentions a dish or groceries, the agent fetches the recipe and price, pays for ingredients through a controlled local wallet, and returns confirmation. Without giving the model the private key. Without classic checkout. With limits, policies, and a log of every payment.**

