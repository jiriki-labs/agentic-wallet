# Jiriki

**Jiriki** is a local wallet daemon for AI agents. Agents never see your private key: they call a small local HTTP API (Unix socket by default) to request payments. Spending is bounded by a YAML policy, recorded in an audit log, and can integrate with **HTTP 402 / x402** payment flows.

This repository contains the Go daemon core, a **Grocery402** demo (recipe lookup and a mock ingredient order behind x402), and project docs.

## Features

- **Policy-gated payments** — per-request caps, daily limits, allowed chains/tokens/merchants, and modes: `dry-run`, `confirm`, or `auto` (see `configs/policy.example.yaml`).
- **x402 client** — thin wrapper around **`github.com/x402-foundation/x402/go`** (V1 exact, EIP-3009) plus facilitator HTTP; suitable for Base Sepolia USDC demos.
- **Local trust boundary** — default listener is a **Unix domain socket** under your config directory; optional TCP with a generated bearer token.
- **Audit trail** — SQLite-backed payment attempt history (`modernc.org/sqlite`, pure Go).
- **Idempotency** — `Idempotency-Key` on payment requests where supported.

## Repository layout

| Path | Description |
|------|-------------|
| `cmd/jiriki/` | CLI: `init`, `up`, `audit`, `policy`, `balance`, `approve`, `version` |
| `internal/` | Daemon implementation: config, keystore, policy, x402, chain helpers, HTTP server, audit store |
| `cmd/grocery-demo/` | Thin stub that points you at the TypeScript demo |
| `apps/grocery/` | **Grocery402** — NestJS API (`api/`) + Next.js UI (`web/`) |
| `configs/` | Example policy YAML |
| `docs/decisions/` | Architecture / technical decision records (ADRs) |
| `AGENTS.md` | Maintainer and agent guidelines (language, scope, verification) |
| `CONTRIBUTING.md` | Commit conventions, Git/PR workflow, agent-related contribution notes |
| `CURSOR_HANDOFF.md` | Phased implementation notes and checklist |

**Go module:** `github.com/jiriki-labs/agentic-wallet` (see `go.mod` for the required Go toolchain version).

## Requirements

- **Go** — version matching `go.mod` (currently `go 1.25.0`).
- **Node.js** and **npm** — only if you run the Grocery402 TypeScript stack under `apps/grocery/`.

## Build and verify (Go)

From the repository root:

```bash
make build          # produces bin/jiriki and bin/grocery-demo (stub)
go test ./... -count=1
go vet ./...
```

If `golangci-lint` is installed:

```bash
make lint
```

## Quick start: wallet daemon

### 1. Initialize a keystore

```bash
make build
./bin/jiriki init
```

You will be prompted for a keystore password. **Do not share** passwords, mnemonics, private keys, or daemon bearer tokens in issues, chat logs, or public screenshots.

### 2. Install a policy file

Copy the example and edit to your needs:

```bash
mkdir -p ~/.config/jiriki
cp configs/policy.example.yaml ~/.config/jiriki/policy.yaml
chmod 700 ~/.config/jiriki
```

Policy fields include allowed tokens/chains/merchants, per-request and daily limits, `requireApprovalAbove`, and `mode` (`dry-run` | `confirm` | `auto`).

### 3. Start the daemon

Default: **Unix socket** at `$XDG_CONFIG_HOME/jiriki/jiriki.sock` (or `~/.config/jiriki/jiriki.sock` on Linux), unless `JIRIKI_HOME` overrides the whole config directory.

```bash
./bin/jiriki up
```

**TCP mode** (optional, e.g. for tools that cannot use Unix sockets):

```bash
./bin/jiriki up --listen-tcp 127.0.0.1:7402
```

On first TCP startup, a bearer token may be written under your config dir — treat it like a secret.

**Flags:**

- `--policy <file>` — policy YAML (default: `~/.config/jiriki/policy.yaml` or under `JIRIKI_HOME`)
- `--facilitator-url <url>` — override the x402 facilitator base URL

**Environment:**

- `JIRIKI_HOME` — config directory (keystore, policy, socket, audit DB, auth token)

## Daemon HTTP API (local)

When the daemon is running, it exposes JSON endpoints such as:

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/balance` | Wallet balance |
| `GET` | `/policy` | Effective policy summary |
| `POST` | `/pay-x402` | Request an x402-gated payment (policy-checked) |
| `GET` | `/transactions` | Recent transaction / attempt history |
| `POST` | `/approve` | Approve a pending payment (when policy requires confirmation) |

Use `Idempotency-Key` on `POST /pay-x402` when your client may retry.

Exact request/response shapes are defined in `internal/daemon/handlers.go`. Default access is via the Unix socket; in TCP mode, authenticate per your deployment (bearer token file created for TCP).

## Grocery402 demo (TypeScript)

The reference merchant + UI live under `apps/grocery/` (Nest API on port **4402**, Next.js dev server on **3020**).

```bash
make grocery-dev
# or: cd apps/grocery && npm install && npm run dev
```

- **API:** `http://127.0.0.1:4402` (override with `PORT`)
- **Web:** `http://127.0.0.1:3020`

The Go binary `bin/grocery-demo` only prints these instructions; it does not start the Node stack.

**Useful environment variables (demo API):**

- `MERCHANT_ADDR` or `GROCERY_MERCHANT_ADDR` — `0x…` recipient for x402 `payTo`
- `GROCERY_SKIP_X402_VERIFY=1` — skip facilitator verification (**local development only**)

Ensure hosts in your policy (`allowedMerchants`) match how you call the API (e.g. `localhost:4402`).

For the agent grocery demo (auto-settle up to **20 USDC**, manual approve above), copy the demo policy instead of the conservative example:

```bash
cp configs/policy.demo.yaml ~/.config/jiriki/policy.yaml
```

Then restart `jiriki up` if the daemon is already running.

## Makefile targets

| Target | Description |
|--------|-------------|
| `make build` | Build `jiriki` and `grocery-demo` into `bin/` |
| `make test` | `go test ./...` |
| `make vet` | `go vet ./...` |
| `make lint` | `golangci-lint run` (skipped if not installed) |
| `make grocery-dev` | Install npm deps and run Nest + Next in dev |
| `make clean` | Remove `bin/` |
| `make demo-dry` / `make demo-mock` | Invoke demo scripts under `scripts/` when present |

If `scripts/` is missing in your clone, those demo targets will fail until the scripts are added or restored.

## Security notes

- This software is **MVP / research-grade**. Do not use funded mainnet keys without understanding the risks.
- Never commit real keystores, bearer tokens, or production endpoints.
- The daemon must not log or leak secrets; if you fork the code, preserve those guarantees.

## Contributing and documentation

- **`CONTRIBUTING.md`** — commit message conventions, Git/PR expectations, and how that applies when using AI coding agents.
- **`AGENTS.md`** — coding principles, stability rules around `internal/`, agent-specific rules, and the verification checklist.
- **`docs/decisions/`** — architectural choices (ADRs).

## License

This project is distributed under the **Beerware License** (Revision 42): you
may do basically anything with it as long as you keep the license notice; if
you ever meet the authors and like the software, tradition says you can buy
them a beer. See [`LICENSE`](LICENSE) for the full text.

Bundled dependencies (Go modules, npm packages, etc.) keep their own licenses.

This is a light-hearted, permissive choice—not legal advice. For corporate use,
have counsel review it like any other license.
