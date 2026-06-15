<p align="center">
  <img src="public/Golden-Toad-logo.webp" alt="money" width="160" />
</p>

<h1 align="center">money</h1>

<p align="center">
  A local-first, self-hostable personal finance backend for AI agents and power users.
</p>

<p align="center">
  <a href="https://github.com/thedavidweng/money/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/thedavidweng/money/ci.yml?branch=main&style=flat-square&label=ci" alt="CI"></a>
  <a href="https://github.com/thedavidweng/money/releases"><img src="https://img.shields.io/github/v/release/thedavidweng/money?style=flat-square" alt="Release"></a>
  <a href="https://github.com/thedavidweng/money/blob/main/LICENSE"><img src="https://img.shields.io/github/license/thedavidweng/money?style=flat-square" alt="License"></a>
  <img src="https://img.shields.io/badge/go-%3E%3D1.26-blue?style=flat-square" alt="Go">
  <a href="https://goreportcard.com/report/github.com/thedavidweng/money"><img src="https://goreportcard.com/badge/github.com/thedavidweng/money?style=flat-square" alt="Go Report"></a>
</p>

---

## What is money?

`money` pulls account and transaction data from user-configured financial providers, stores it in a user-owned **encrypted SQLite database**, and exposes stable **CLI + JSON contracts** for external agents, scripts, and cron jobs.

It does not embed AI chat, model providers, hosted billing, telemetry, or a required long-running server. Your data stays local. Your agent owns the reasoning.

## Why

Existing personal finance tools either lock data behind a paid SaaS, embed opinionated AI advisors, or assume a full web-app product shape. `money` takes a different approach:

- **Local-first data ownership** — your financial data lives in an encrypted file you control.
- **Agent-friendly contracts** — stable JSON envelopes that any AI agent, script, or automation can parse.
- **Provider-neutral** — Plaid, Bridge, CSV imports — providers are replaceable adapters.
- **No server required** — runs as a CLI on your laptop, in cron, or in CI.

## Getting Started

New to `money`? Read the full [Getting Started guide](docs/GETTING_STARTED.md) for step-by-step setup and pricing details.

### Quick Start

```bash
# macOS/Linux
curl -fsSL https://raw.githubusercontent.com/thedavidweng/money/main/install.sh | sh

money setup
money link
money sync
money accounts list --json
money transactions search "Costco" --json
```

The installer detects Homebrew automatically and uses the cask when available. Otherwise it downloads the release binary to `~/.local/bin`.

<details>
<summary>Other installation methods</summary>

**Windows PowerShell:**

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/thedavidweng/money/main/install.ps1 | iex"
```

The latest release must include a Windows archive for this installer to complete. If it does not, the script fails explicitly and points you to `go install`.

**Homebrew Cask (macOS/Linux):**

```bash
brew install --cask thedavidweng/tap/money
```

If you installed an older Homebrew formula release, migrate to the cask:

```bash
brew update
brew uninstall --formula thedavidweng/tap/money
brew install --cask thedavidweng/tap/money
money version
```

**Go:**

```bash
go install github.com/thedavidweng/money/cmd/money@latest
```

**Manual download:** grab the archive for your platform from the [latest GitHub Release](https://github.com/thedavidweng/money/releases/latest), extract it, and place the `money` binary on your `PATH`.

</details>

Try it without real credentials:

```bash
money demo accounts list --json
money demo transactions search "coffee" --json
```

### Uninstall

```bash
# Homebrew Cask
brew uninstall --cask thedavidweng/tap/money

# install.sh
curl -fsSL https://raw.githubusercontent.com/thedavidweng/money/main/install.sh | sh -s uninstall

# Go
rm "$(go env GOPATH)/bin/money"
```

Your local `~/.money` config, secrets, and database are not removed by uninstalling.

## Architecture

```text
cmd/money/             CLI entrypoint (Cobra)
internal/cli/          CLI commands and doctor diagnostics
internal/config/       Configuration loading and validation
internal/contracts/    JSON envelopes, schema versions, error codes
internal/core/         Finance primitives and domain types
internal/importsource/ CSV and Monarch Money importers
internal/linking/      Plaid Link helper functions
internal/plaidlogin/   Plaid Dashboard OAuth login flow
internal/prompt/       Interactive TUI prompts (via Charm huh)
internal/providers/    Provider adapters (Plaid, Bridge, …)
internal/store/        Encrypted SQLite store and migrations
internal/syncer/       Transaction sync orchestration
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
| [`docs/COMMANDS.md`](docs/COMMANDS.md) | Command inventory and global flags |
| [`docs/PRD.md`](docs/PRD.md) | Product requirements |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Module boundaries and data flow |
| [`docs/CONTRACTS.md`](docs/CONTRACTS.md) | Current CLI JSON contracts |
| [`docs/SCHEMA.md`](docs/SCHEMA.md) | Database schema contract |
| [`docs/CONFIG.md`](docs/CONFIG.md) | Configuration loading rules |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | Development phases |
| [`CHANGELOG.md`](CHANGELOG.md) | Release history |
| [`SECURITY.md`](SECURITY.md) | Vulnerability reporting policy |

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


## Infrastructure

- **CI/CD:** [cli-workflow-template](https://github.com/thedavidweng/cli-workflow-template) — reusable GitHub Actions workflows
- **Docs:** [site](https://github.com/thedavidweng/site) — landing page and documentation

## License

[Apache 2.0](LICENSE)
