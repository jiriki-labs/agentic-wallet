// This directory hosts the TypeScript Grocery402 app (npm workspaces).
// A nested go.mod prevents the parent module's `go test ./...` from
// descending into node_modules (e.g. flatted's golang shim).
module github.com/jiriki-labs/agentic-wallet/apps/grocery

go 1.25.0
