# AGENTS.md — Jiriki (agentic-wallet)

Guidance for humans and coding agents working on this repository.

## Language policy

**All project documentation MUST be written in English.** That includes new docs under `docs/`, README updates, ADRs, code comments meant as public API documentation, and agent-facing instructions in skills. Informal notes may exist in other languages historically; treat English as the canonical project language going forward.

---

## Core commandments

### KISS (Keep It Simple, Stupid)

- Prefer the smallest change that solves the problem.
- Avoid new abstractions until a second real use case appears.
- Keep binaries, packages, and flags easy to reason about for an MVP wallet daemon.
- Favor straightforward Go over clever patterns.

### DRY (Don't Repeat Yourself)

- Extract shared logic only when duplication is real and stable—not speculative.
- Do not copy-paste security-sensitive flows (signing, policy checks, x402); reuse existing modules.
- When you fix a bug, fix the single source of truth; do not scatter parallel implementations.

---

## What this project is

**Jiriki** is a local, lightweight **wallet daemon** for AI agents. Agents never receive the private key; they call a controlled local API (Unix socket by default) to request payments under YAML policy limits. Payments integrate with **HTTP 402 / x402** style flows, audit logging, and optional human confirmation.

**Demo (MVP):** **Grocery402** — recipe lookup plus ingredient ordering behind x402, implemented as `grocery-demo` (see `CURSOR_HANDOFF.md`). The wallet daemon in `internal/` and `cmd/jiriki/` is the stable core; the demo merchant and Claude skill live at the repo edge.

---

## Repository map (high level)

| Area | Role |
|------|------|
| `cmd/jiriki/` | CLI and daemon entrypoints (`up`, `init`, policy, audit, balance, approve, …) |
| `internal/` | Daemon core: config, keystore, policy, x402 client, chain helpers, daemon HTTP, audit store |
| `cmd/grocery-demo/` | Demo merchant HTTP server (x402-gated orders; stub until Phase 5 is complete) |
| `configs/` | Example policy YAML |
| `docs/decisions/` | ADRs and technical decisions |

**Stability rule:** `internal/` is the trusted implementation of signing, policy, x402, and persistence. Do **not** refactor or “clean up” `internal/` unless a task explicitly requires it; prefer thin changes at the edges (`cmd/`, new demo packages, docs).

---

## Engineering principles

1. **Security first:** agents must not see private keys, bearer tokens for the daemon, or keystore passwords in logs or model context.
2. **Policy-gated payments:** every payment path goes through the policy engine; defaults should favor dry-run or confirm over silent auto-spend.
3. **Local trust boundary:** daemon listens on **Unix socket** by default; TCP is optional for portability.
4. **Observable attempts:** payment attempts are recorded (audit); MVP may use plain SQLite without tamper-evidence—do not silently weaken guarantees without an ADR.
5. **Idempotency:** respect `Idempotency-Key` and existing idempotency semantics; do not break deduplication.
6. **MVP scope:** avoid browser extensions, mobile wallets, multi-tenant SaaS, and full consumer wallet UX unless the roadmap explicitly expands.

---

## Tech stack (authoritative shortcuts)

- **Language:** Go (see `go.mod` for the toolchain version).
- **Module:** `github.com/jiriki-labs/agentic-wallet`.
- **Ethereum / keys:** `go-ethereum` keystore (scrypt).
- **SQLite:** `modernc.org/sqlite` (pure Go, no CGO).
- **x402:** `internal/x402` wraps **`github.com/x402-foundation/x402/go`** (V1 exact + facilitator HTTP); see `docs/decisions/0001-x402-client.md`.
- **Default chain (demo):** Base Sepolia; USDC contract as documented in handoff/policy examples.

---

## Code style

- **Formatting:** `gofmt` / standard Go conventions.
- **Indentation:** tabs for Go (`.editorconfig`); YAML uses 2 spaces.
- **Linting:** `.golangci.yml` enables `errcheck`, `gosimple`, `govet`, `ineffassign`, `staticcheck`, `unused`; errcheck skips `_test.go` per config.
- **Tests:** table-driven where appropriate; keep tests close to packages they cover.

---

## Verification (run before claiming done)

```bash
go test ./... -count=1
go vet ./...
make build
```

When `golangci-lint` is installed:

```bash
make lint
```

Demo targets (when scripts exist and environment is configured):

```bash
make demo-dry
make demo-mock
```

---

## Agent-specific rules

- **Never log or echo:** daemon bearer token, keystore password, mnemonic, or raw private key material.
- **Never instruct users to paste secrets** into chat or commit them to git.
- **Minimal diffs:** change only what the task requires; no drive-by refactors or unrelated formatting sweeps.
- **Documentation:** new or updated **project** documentation in **English** only (this file, `docs/`, README, skills intended for agents).
- **Secrets:** use `configs/policy.example.yaml` style examples with placeholder addresses; do not commit real funded keys or production endpoints without explicit project process.

---

## Working with coding agents (humans + tools)

Use this file as the **source of truth** for agent behavior in this repo. When using Cursor, Claude Code, or similar:

1. **Point the agent at this file** (many tools auto-read `AGENTS.md` in the workspace root).
2. **State the task narrowly** — which package, which behavior, what “done” means; remind agents not to refactor `internal/` unless required.
3. **Review all agent output** before merge: policy and signing paths are security-sensitive.
4. **Git:** if the agent creates commits, require **English** messages and the conventions in **`CONTRIBUTING.md`** (Conventional Commits–style prefixes, imperative subject, no secrets in diff or message).

Optional: keep local daemon data **outside** the clone (set `JIRIKI_HOME` to a directory that is not the repository) so keystores and `auth` never appear beside source code.

---

## Git commits (when you or an agent commit)

- Follow **`CONTRIBUTING.md`** for message format, PR scope, and pre-push verification.
- Prefer **one logical change per commit**; squash only when it improves clarity.
- Never commit ignored artifacts: see **`.gitignore`** (build outputs, Node artifacts, local env files, Jiriki runtime files).

---

## Product one-liner (for context)

> A local wallet daemon that lets AI agents request payments within user-defined limits, with policy enforcement and auditability—without giving the model the private key.

---

## Where to look next

- `README.md` — public overview, build, and demo quick start.
- `CONTRIBUTING.md` — commit conventions, Git/PR expectations, and agent-related contribution notes.
- `CORE_CONCEPT.md` — deeper product narrative (legacy language; English ADRs and README should converge over time).
- `docs/decisions/` — recorded technical choices (keep new ADRs in English).
