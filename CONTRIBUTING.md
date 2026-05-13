# Contributing to Jiriki

Thank you for helping improve this project. This document covers **Git usage**, **commit conventions**, and how that fits together with **AI coding agents** (see also `AGENTS.md`).

---

## Before you open a PR

1. Run the checks in `AGENTS.md` (Verification section), at minimum:

   ```bash
   go test ./... -count=1
   go vet ./...
   make build
   ```

2. For Go changes, run `make lint` when `golangci-lint` is installed.

3. For the Grocery402 TypeScript stack (`apps/grocery/`), run the workspace’s `lint` / `test` scripts if you changed that code.

4. Do **not** commit keystores, bearer tokens, mnemonics, `.env` files with secrets, or real production endpoints.

---

## Commit messages

Use **clear, complete English** in the subject line (imperative mood, as if completing “This commit will …”).

**Recommended style (Conventional Commits):** optional type prefix + short description.

| Prefix | Use for |
|--------|---------|
| `feat:` | New user-visible behavior or API |
| `fix:` | Bug fix |
| `docs:` | Documentation only |
| `test:` | Tests only |
| `chore:` | Tooling, deps, build scripts without functional change |
| `refactor:` | Internal restructuring without behavior change |

**Examples:**

- `fix: reject pay-x402 when merchant host not in policy`
- `docs: add CONTRIBUTING and tighten gitignore`
- `chore: bump go-ethereum patch version`

**Body (optional):** explain *why* the change was needed if the subject is not enough. Link issues with `Fixes #123` when applicable.

**Avoid:** vague subjects (`update`, `fix stuff`, `wip`), huge unrelated bundles in one commit, or secrets in message or diff.

---

## Branching and pull requests

- Prefer **small, reviewable PRs** scoped to one concern when possible.
- **Rebase or merge** according to maintainer preference on the default branch; keep history readable and conflict-free before review.
- In PR descriptions, summarize **what** changed and **why**, list test commands you ran, and call out any security or policy-behavior changes.

---

## Working with AI coding agents (Cursor, Claude Code, etc.)

Coding agents should follow **`AGENTS.md`** end-to-end. In short:

- **Security:** never log or paste daemon bearer tokens, keystore passwords, mnemonics, or private keys; do not commit files under `.gitignore` “secrets and local runtime” patterns.
- **Scope:** minimal diffs; do not refactor `internal/` unless the task explicitly requires it.
- **Language:** new project documentation and comments intended as public API docs stay in **English**.
- **Verification:** run the same Go checks as above before claiming work is done.

When an agent is asked to **create Git commits**, it should still follow this file’s **commit message** rules and keep commits **atomic** (one logical change per commit when practical).

Humans remain responsible for reviewing agent-produced changes before merge.

---

## Where to get help

- `README.md` — setup and high-level architecture
- `AGENTS.md` — agent rules, stability boundaries, verification
- `docs/decisions/` — ADRs for deeper technical context
