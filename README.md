# money

`money` is a local-first, self-hostable personal finance backend for AI agents and power users.

The project is intentionally small at the core: pull account and transaction data from user-configured providers, store it in a user-owned encrypted SQLite database, and expose stable CLI + JSON contracts for external agents. It does not embed AI chat, model providers, hosted billing, telemetry, or a required long-running server.

## First Principles

- Local-first data ownership.
- CLI-first access for humans, scripts, and AI agents.
- Stable JSON contracts over human terminal formatting.
- Provider adapters behind a small finance core.
- Explicit failures instead of hidden fallback behavior.
- Small, deterministic modules that are easy to test.

## Initial Shape

```text
cmd/money/             CLI entrypoint
internal/contracts/    JSON envelope and schema versions
internal/core/         finance primitives and domain types
internal/providers/    provider interfaces and adapters
internal/store/        SQLite store and migrations
docs/                  product, architecture, and reference notes
donors/                optional local reference clones, ignored by git
```

## Acknowledgements

- `monarchmoney-cli`: agent-friendly CLI contracts and safety model.
- `ray-finance`: Plaid/local database/sync/import ideas.
- `actual`: local-first budgeting and automation patterns.
- `maybe`: personal finance domain modeling and product vocabulary.

## Current Status

Bootstrap only. The first implementation target is a small read-only contract surface:

```bash
money accounts list --json
money transactions list --json
money transactions search "Amazon" --json
money categories list --json
money tags list --json
money recurring list --json
```
