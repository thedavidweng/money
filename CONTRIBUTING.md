# Contributing

Thanks for your interest in contributing to `money`.

## Project Philosophy

`money` is a **local-first, agent-friendly** personal finance backend. Contributions should align with these principles:

- **Stable JSON contracts above all.** Breaking changes to the contract must be intentional and versioned.
- **Keep the core boring.** Simple, deep, deterministic modules over clever abstractions.
- **No AI chat, model providers, hosted billing, telemetry, or required server behavior in the core.**
- **Provider adapters are replaceable.** Provider-specific logic lives behind interfaces.

## Development

```bash
git clone https://github.com/thedavidweng/money.git
cd money
go mod download

# Run tests
go test ./...

# Build
go build ./cmd/money
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
