<p align="center">
  <img src="public/Golden-Toad-logo.webp" alt="money" width="160" />
</p>

<h1 align="center">money</h1>

<p align="center">
  A local-first, self-hostable personal finance backend for AI agents and power users.
</p>

<p align="center">
  <a href="https://github.com/thedavidweng/money/actions"><img src="https://img.shields.io/github/actions/workflow/status/thedavidweng/money/ci.yml?branch=main&style=flat-square" alt="CI"></a>
  <a href="https://github.com/thedavidweng/money/releases"><img src="https://img.shields.io/github/v/release/thedavidweng/money?style=flat-square" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/thedavidweng/money?style=flat-square" alt="License"></a>
  <img src="https://img.shields.io/badge/go-%3E%3D1.25-blue?style=flat-square" alt="Go">
</p>

---

## What is money?

`money` pulls account and transaction data from user-configured financial providers, stores it in a user-owned **encrypted SQLite database**, and exposes stable **CLI + JSON contracts** for external agents, scripts, and cron jobs.

It does not embed AI chat, model providers, hosted billing, telemetry, or a required long-running server. Your data stays local. Your agent owns the reasoning.

## Why

Existing personal finance tools either lock data behind a paid SaaS, embed opinionated AI advisors, or assume a full web-app product shape. `money` takes a different approach:

- **Local-first data ownership** — your financial data lives in an encrypted file you control.
- **Agent-friendly contracts** — stable JSON envelopes that any AI agent, script, or automation can parse.
- **Provider-neutral** — Plaid, Bridge, MX, Finicity, CSV imports — providers are replaceable adapters.
- **No server required** — runs as a CLI on your laptop, in cron, or in CI.

## Getting Started

👉 **New to money?** Read the full [Getting Started guide](docs/GETTING_STARTED.md) for step-by-step installation, setup, and pricing details.

### Quick Start

```bash
# Install via Homebrew (macOS/Linux)
brew install thedavidweng/tap/money

# Or install via Go (cross-platform)
go install github.com/thedavidweng/money/cmd/money@latest

# Initialize configuration and encrypted database (interactive)
money setup

# Link your bank and sync data
money link
money sync

# Query your local data
money accounts list --json
money transactions search "Costco" --json
```

Try it without real credentials:

```bash
money demo accounts list --json
money demo transactions search "coffee" --json
```

## Commands

```text
# Getting Started
money setup                         Initialize configuration and encrypted database
money doctor                        Check configuration and system health
money demo <command>                Run against non-persistent sample data

# Data Queries
money accounts list                 List financial accounts
money accounts create-manual        Create a local manual account
money transactions list             List transactions with filters
money transactions search           Search transactions by text
money categories list               List transaction categories
money tags list                     List transaction tags
money recurring list                List recurring transactions
money investments holdings          List investment holdings
money investments securities        List investment securities
money liabilities list              List liabilities
money items list                    List linked provider items
money items get <id>                Get a linked provider item
money items rename <id> <name>      Rename a provider item alias
money items remove <id>             Remove a linked provider item with cascade delete

# Provider Management
money link                          Link a financial institution
money providers configure <provider> Configure provider credentials
money providers plaid link          Link a Plaid Provider Item
money providers bridge link         Link a Bridge Provider Item
money sync                          Sync linked provider data (supports --start-date/--end-date)

# Utilities
money feedback                      Open the project GitHub issues page
money version                       Print version
```

### Global Flags

```text
--config string            config file path
--profile string           configuration profile (default "default")
-j, --json                 write a JSON envelope to stdout
```

Read commands and provisional sync diagnostics support `--json` for machine-readable output. Manual write operations require `--dry-run` or `--confirm`.

## Architecture

```text
cmd/money/             CLI entrypoint (Cobra)
internal/contracts/    JSON envelopes, schema versions, error codes
internal/core/         Finance primitives and domain types
internal/providers/    Provider adapters (Plaid, Bridge, …)
internal/store/        Encrypted SQLite store and migrations
internal/config/       Configuration loading and validation
```

Read commands use local data only. Sync is the explicit boundary where outbound provider calls happen. See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full design.

## Design Principles

- **CLI-first** — human output by default, `--json` for automation.
- **Stable contracts** — versioned JSON envelopes with deterministic sorting and pagination.
- **Explicit failures** — no hidden fallbacks, no silent downgrades.
- **BYOK providers** — bring your own Plaid/Bridge credentials; no managed proxy or subscription.
- **Encrypted at rest** — real financial data never touches plaintext SQLite.
- **Small deep modules** — simple interfaces over pass-through wrappers.

## Documentation

| Document | Purpose |
|----------|---------|
| [`docs/GETTING_STARTED.md`](docs/GETTING_STARTED.md) | Install, setup, and pricing guide |
| [`docs/PRD.md`](docs/PRD.md) | Product requirements |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Module boundaries and data flow |
| [`docs/CONTRACTS.md`](docs/CONTRACTS.md) | Current CLI JSON contracts |
| [`docs/SCHEMA.md`](docs/SCHEMA.md) | Database schema contract |
| [`docs/CONFIG.md`](docs/CONFIG.md) | Configuration loading rules |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | Development phases |

## Website

The project landing page (Astro, static HTML) lives in [`website/`](website/). Published site: **https://thedavidweng.github.io/money/** (enable **GitHub Pages** → GitHub Actions in the repository settings if it is not live yet).

```bash
cd website
npm ci
npm run dev    # local preview
npm run build  # output in website/dist
```

## Acknowledgements

`money` draws inspiration from several excellent projects:

- [**monarchmoney-cli**](https://github.com/thedavidweng/monarchmoney-cli) — agent-friendly CLI contract design, JSON envelope patterns, and safety model.
- [**Ray Finance**](https://github.com/cdinnison/ray-finance) — Plaid/Bridge sync architecture, local encrypted database patterns, and provider adapter design.
- [**Actual Budget**](https://github.com/actualbudget/actual) — local-first budgeting philosophy and automation API patterns.
- [**Maybe Finance**](https://github.com/maybe-finance/maybe) — personal finance domain modeling and product vocabulary.

## License

[MIT](LICENSE)
