# Contributing

Thanks for your interest in contributing to `money`.

## Project Philosophy

`money` is a **local-first, agent-friendly** personal finance backend. Contributions should align with these principles:

- **Stable JSON contracts above all.** Breaking changes to the contract must be intentional and versioned.
- **Keep the core boring.** Simple, deep, deterministic modules over clever abstractions.
- **No AI chat, model providers, hosted billing, telemetry, or required server behavior in the core.**
- **Provider adapters are replaceable.** Provider-specific logic lives behind interfaces.

## Development

### Prerequisites

- [Go 1.26](https://go.dev/dl/) or later (see `go.mod` for the exact version)

### Setup

```bash
git clone https://github.com/thedavidweng/money.git
cd money
mise install  # install tools pinned in mise.toml
go mod download
```

### Build and Test

```bash
# Format check (CI will fail if any files are unformatted)
test -z "$(gofmt -l .)"

# Static analysis
go vet ./...

# Run tests with race detector (matches CI)
go test ./... -race

# Run tests with coverage summary
go test ./... -cover

# Build the binary
go build ./cmd/money
```

CI runs `gofmt -l`, `go vet`, `go test -race`, and `go build` on both `ubuntu-latest` and `macos-latest` for every push and pull request.

### Release Workflow

Releases are automated via [GoReleaser](https://goreleaser.com/) and triggered by pushing a `v*` tag. The release workflow requires a `HOMEBREW_TAP_GITHUB_TOKEN` secret with push access to the Homebrew tap repository. To test the release process locally:

```bash
goreleaser release --snapshot --clean
```

## Pull Requests

1. Open an issue first to discuss the change.
2. Keep PRs focused — one concern per PR.
3. Update documentation (`docs/`, `README.md`, `AGENTS.md`) alongside code changes.
4. Follow existing code style and Go conventions.
5. Add tests for new behavior.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add Plaid provider adapter
fix: handle null category in envelope serialization
docs: clarify encryption key rotation
```

## Code of Conduct

Be respectful. This is a personal project but contributors are welcome.
