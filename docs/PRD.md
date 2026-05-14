# Product Requirements Document: money

## Problem Statement

The current personal finance workflow depends on Monarch Money as the source of truth. That works today, but it is not sustainable if the user wants local-first data ownership, provider choice, and a stable interface for AI agents. Existing open-source finance products contain useful pieces, but they either embed AI behavior, assume a full app/server product shape, focus on budgeting UX, or carry licensing and architecture choices that do not fit a small local backend.

The user needs a local-first, self-hostable personal finance backend that can gradually replace Monarch-dependent workflows while preserving the agent-friendly command semantics that made `monarchmoney-cli` valuable.

## Solution

Build `money`: a Go-first local finance backend and CLI that syncs data from user-configured financial providers into an encrypted SQLite database, then exposes deterministic finance primitives through stable JSON contracts.

The first version should run on a laptop without a persistent server. It should support explicit BYOK provider sync, read from a user-owned encrypted local database, and return machine-readable envelopes suitable for AI agents. AI reasoning, memory, chat, planning, and recommendations stay outside the product.

The first stable command contracts are:

1. `money accounts list --json`
2. `money transactions list --json`
3. `money transactions search <query> --json`
4. `money categories list --json`
5. `money tags list --json`
6. `money recurring list --json`
7. `money accounts create-manual --json`

The concrete JSON schema notes, examples, sync result shape, and error taxonomy are maintained in `docs/CONTRACTS.md`.

After these contracts are proven, the project can add budgets, spending summaries, cashflow, rules, annotation write operations, more providers, and imports. Provider sync, demo mode, and manual account creation are first-version capabilities; Monarch/CSV/Apple Card imports are migration work and do not block the first stable read contracts.

## User Stories

1. As a local-first finance user, I want my account and transaction data stored in my own encrypted SQLite database, so that I am not locked into a paid finance SaaS or plaintext local files.
2. As an AI agent user, I want stable JSON output, so that my agent can parse finance data reliably.
3. As an AI agent user, I want finance commands to avoid embedded AI behavior, so that my external agent owns reasoning and memory.
4. As a Monarch user, I want equivalent command semantics in `money`, so that my existing agent workflow can migrate gradually.
5. As a provider-conscious user, I want Plaid and Bridge to be adapters, so that future providers can replace or supplement them.
6. As a user linking a bank, I want to start from the institution I use and choose among supported Providers when more than one can connect it.
7. As a privacy-conscious user, I want explicit outbound calls only during sync/link operations, so that read commands do not unexpectedly send data elsewhere.
8. As a CLI user, I want commands to work without a long-running server, so that the tool feels like local infrastructure rather than an app stack.
9. As a developer, I want provider data normalized into a canonical schema, so that commands stay stable when data sources change.
10. As a developer, I want deterministic sorting and pagination, so that agent results are repeatable.
11. As a developer, I want explicit errors instead of hidden fallback behavior, so that failures are diagnosable.
12. As a finance automation user, I want cron jobs and external agents to call the same CLI contracts, so that automation does not depend on a long-running service.
13. As a finance data owner, I want migration/import paths from Monarch and CSV sources, so that historical data can survive provider changes.
14. As a user with multiple account types, I want checking, savings, credit, loans, investments, property, and manual accounts modeled consistently, so that net worth and cashflow can be calculated locally.
15. As a budgeting user, I want budgets and rules to be deterministic data primitives, so that agents can inspect and modify them through approvals.
16. As a safety-conscious user, I want write operations to support dry-run and confirmation gates, so that agents cannot mutate finance data accidentally.
17. As a BYOK user, I want to provide financial provider credentials locally without registering a `money` account or subscription, so that sync remains under my control.
18. As a CLI user, I want to configure `money` through `~/.money/config.yaml`, `.env`, environment variables, or setup commands, so that it fits both interactive and automated local workflows.
19. As a user with overlapping Provider coverage, I want duplicate-looking linked data to stay separate until I explicitly review and approve a merge.
20. As a Plaid BYOK user, I want an optional Plaid Dashboard login bootstrap command, so that local API keys can be fetched into my resolved `.env` without making `money` a hosted credential proxy.

## Implementation Decisions

- The primary implementation language is Go.
- The primary storage engine is encrypted SQLite.
- The CLI is the first supported interface.
- CLI routing uses Cobra.
- Encrypted SQLite uses `github.com/ncruces/go-sqlite3` with the encrypted `adiantum` VFS for real stores.
- JSON contracts are the source of truth for automation.
- Human terminal formatting is secondary.
- The finance core owns accounts, institutions, transactions, budgets, rules, sync results, and query logic.
- Provider adapters own communication with external financial Providers and map provider data into canonical records.
- Provider support is BYOK-only; `money` does not require registered accounts, subscriptions, managed proxy credentials, or AI API keys.
- The first networked Providers are Plaid and Bridge.
- Plaid Dashboard login is a credential-bootstrap convenience for local Plaid API keys. It is not a general hosted account model and must preserve manual `money providers configure plaid` as the stable fallback.
- Providers are peers; users choose based on institution support, not a global provider hierarchy.
- Linking the same institution through multiple Providers creates separate Provider Items.
- Merging or deduplicating Provider Items or accounts requires explicit user review and approval.
- MX and Finicity are later Provider targets.
- CSV, Apple Card export, and Monarch export are Import Sources, not Providers.
- Early read contracts include categories, tags, recurring transactions, and transaction filters for tags and recurring status. Split transaction support is deferred until a local split schema and write safety model exist.
- Manual accounts are part of the first implementation so `money` can be useful offline and support demo/test data without a Provider.
- Local configuration belongs to `money`; it must not fall back to Ray, Monarch, or donor config paths or environment variable names.
- Config sources complete each other through explicit references instead of silently overriding one another.
- The project should not embed AI chat, LLM providers, conversation memory, hosted billing, telemetry, or a required web dashboard.
- Donor repositories are reference material only and are kept out of the project git history.
- Ray Finance is the primary engineering donor for local schema, Provider sync, imports, local annotations, and query fields.
- `monarchmoney-cli` is the primary donor for agent-friendly CLI contract design, envelopes, exit codes, and safety gates.
- Actual Budget is the primary donor for local-first budgeting and automation lessons.
- Maybe Finance is a donor for product and domain modeling ideas, but its Rails/PostgreSQL/AGPL shape is not the foundation.

## Testing Decisions

- Tests should target external behavior: command output, contract shape, provider mapping, store queries, and migration behavior.
- Contract tests are required before a command is considered stable.
- Tests use Go's standard `testing` package by default. Add `testify` only if assertion boilerplate starts hiding test intent.
- Provider adapters should be tested with fixture responses, not live provider calls.
- Store tests should run against temporary encrypted SQLite databases.
- CLI tests should verify stdout/stderr separation for JSON output.
- Error tests should assert stable machine-readable error codes.
- Safety tests should cover dry-run, confirmation, and read-only behavior before mutation commands are introduced.

## Out of Scope

- Embedded AI advisor or chat UI.
- Long-running required server.
- Hosted cloud backend.
- Subscription billing.
- Telemetry or analytics.
- Full dashboard.
- Plugin runtime.
- Dynamic provider installation.
- Write-heavy schema surface before the initial read contracts are stable.
- Direct reuse of AGPL donor code unless the project intentionally adopts compatible licensing.
- Plaintext SQLite storage for real financial data.

## Further Notes

The project should follow Occam's razor and first principles: build the smallest useful local finance backend, avoid fallback behavior, and regularly check for dead or obsolete code. The first release should prove that an external agent can list accounts, list recent transactions, and search transactions through stable contracts independent of Monarch.
