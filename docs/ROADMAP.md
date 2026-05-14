# Roadmap

## Phase 0: Bootstrap

- Create project skeleton.
- Capture vision, PRD, donor notes, and architecture.
- Keep donor repositories isolated under `donors/`.
- Verify the empty CLI builds.

## Phase 1: First Contracts

- Implement `internal/config` using `docs/CONFIG.md`.
- Implement encrypted SQLite migrations for institutions, provider items, accounts, transactions, categories, tags, recurring items, and sync runs.
- Keep first migration shape aligned with `docs/SCHEMA.md`.
- Implement `money accounts list --json`.
- Implement `money transactions list --json` and `money transactions search <query> --json`, with `tx` aliases.
- Implement `money categories list --json`, `money tags list --json`, and `money recurring list --json`.
- Add contract tests for first-milestone read commands.
- Add deterministic sorting and pagination.

## Phase 2: Plaid and Bridge Sync

- Add Plaid configuration.
- Add Plaid Dashboard login as an optional credential-bootstrap path while preserving manual BYOK configuration.
- Add Bridge configuration.
- Add explicit provider link/sync commands.
- Store Plaid and Bridge item, account, and transaction data in canonical tables.
- Add provider fixture tests.

## Phase 3: Monarch Migration

- Add Monarch Import Source.
- Map Monarch accounts and transactions into canonical records.
- Preserve agent-facing command semantics from `monarchmoney-cli`.

## Phase 4: Finance Primitives

- Budgets.
- Categories.
- Rules.
- Cashflow.
- Spending summaries.
- Recurring bills.
- Net worth.

## Phase 5: Additional Local Primitives

- Additional providers such as MX and Finicity.
- Additional Import Sources such as CSV and Apple Card export.
- Additional command contracts for local automation.
