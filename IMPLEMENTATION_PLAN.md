# Implementation Plan: Provider Support

This plan is written for an agent with no prior memory of the conversation. Read this file first, then follow the referenced project documents before editing code.

## Goal

Implement BYOK financial Provider support for `money`, starting with Plaid and Bridge, while preserving the useful finance boundaries captured in the Ray donor documents without carrying over OpenRay naming, managed SaaS behavior, or AI-key requirements.

`money` must become a local-first, CLI-first finance backend for external AI agents:

- Sync user-configured provider data into a user-owned encrypted SQLite database.
- Expose finance primitives through stable CLI + JSON contracts.
- Keep AI reasoning, chat memory, model providers, hosted billing, telemetry, and dashboards outside the core.
- Keep providers replaceable: Plaid and Bridge first, later MX and Finicity.
- Keep file and migration inputs separate as Import Sources: Monarch export, CSV, and Apple Card export.
- Support manual accounts as local, offline-first data.
- Support an isolated non-persistent demo environment that never writes to the real store.
- Require user-provided Provider credentials locally; do not require registered accounts, subscriptions, managed proxy keys, or AI API keys.
- Rewrite in Go. Donor code is reference material only.

## Required Reading

Read these files before implementation:

1. `AGENTS.md`
2. `docs/PRD.md`
3. `docs/ARCHITECTURE.md`
4. `docs/CONFIG.md`
5. `docs/SCHEMA.md`
6. `docs/DONORS.md`
7. `donors/ray-finance/CONTEXT.md`
8. `donors/ray-finance/ARCHITECTURE.md`

The Ray OpenRay documents are especially important. They define the product language and boundaries that `money` should preserve:

- A **Finance Primitive** is a stable operation over local financial data.
- A **Command Contract** is a versioned machine-readable promise for CLI input/output.
- An **External Agent** owns reasoning, memory, and planning outside `money`.
- A **Provider** is a live networked financial integration that links and syncs institution data into the local store.
- An **Import Source** is a file or migration input, such as Monarch export, CSV, or Apple Card export.
- A **Sync** updates local data only.
- Read commands must not require AI keys or provider credentials.
- Provider commands require provider credentials and should fail explicitly when missing.

When command, data model, or Provider behavior can be answered by donor code, prefer donor evidence before asking product questions. Use Ray Finance as the primary engineering donor for local data model decisions, Provider sync, Plaid/Bridge flow, encrypted local storage behavior, imports, local annotations, and query fields. Use `monarchmoney-cli` for command naming habits, JSON envelopes, stdout/stderr behavior, exit codes, safety gates, and agent-facing ergonomics. If Ray lacks a feature, do not invent a first-version stable contract from Monarch alone; defer it unless it is required by `money`'s goals. Ask only when donor behavior conflicts with `money` boundaries or a real product trade-off remains.

Decision ownership: engineering implementation details are decided by the implementer using best engineering and security practices, donor evidence, and `money` constraints. Ask the user only for user experience, product boundary, risk acceptance, naming/workflow, or interoperability decisions.

## Donor Policy

Donors live in `donors/`, which is ignored by git. Do not commit donor source code.

Use donors this way:

- Read donor implementation to understand provider flow, edge cases, and field mapping.
- Rewrite the implementation in Go using `money`'s modules and naming.
- Do not copy donor files, comments, fixtures, schemas, tests, generated assets, or exact implementation text.
- When a donor materially informs behavior, update `README.md` acknowledgements or `docs/DONORS.md`.
- Maybe is AGPL. Do not copy code from Maybe unless the project intentionally adopts a compatible license.

## Donor Reference Map

### Ray Finance / OpenRay

Primary reference for external provider sync and OpenRay boundaries.

Read:

- `donors/ray-finance/CONTEXT.md`
- `donors/ray-finance/ARCHITECTURE.md`
- `donors/ray-finance/src/providers/types.ts`
- `donors/ray-finance/src/providers/index.ts`
- `donors/ray-finance/src/providers/plaid.ts`
- `donors/ray-finance/src/plaid/client.ts`
- `donors/ray-finance/src/plaid/link.ts`
- `donors/ray-finance/src/plaid/sync.ts`
- `donors/ray-finance/src/db/schema.ts`
- `donors/ray-finance/src/db/encryption.ts`
- `donors/ray-finance/src/daily-sync.ts`

Extract these ideas:

- Static provider registry for the first version.
- Provider interface with configured-state checks, link, and sync.
- Institution-first linking flow where Providers are peers and the user chooses among Providers that support the selected institution.
- Plaid Link token creation and public-token exchange.
- Plaid item, account, transaction, investment, liability, and recurring sync flow.
- Bridge client, connect session, item status, reconnect state, pagination, and account/transaction mapping.
- Sync cursors for incremental transaction sync.
- Monarch-style canonical transaction amount semantics: positive inflow, negative outflow.
- Canonical account balance semantics: assets positive, liabilities negative; credit limits and available credit are separate fields; available balance is cash-like only.
- App-managed encrypted SQLite storage for local financial data, implemented with the best fit for Go single-binary distribution and current security practice.
- Direct BYOK Provider configuration.
- Post-sync derived work should be explicit and isolated.

Do not carry over:

- `ray` names, `~/.ray`, `RAY_*`, `RAY_API_KEY`, `RAY_PROXY_BASE`, or hosted Ray proxy behavior.
- Managed setup or billing.
- Managed Provider proxy behavior.
- `src/ai/*`, chat, prompts, model providers, memories, AI audit logs, or PII redaction for LLM prompts.
- The Plaid Link helper server as a general local API. If used, it is only a short-lived linking helper.
- Silent fallbacks around missing config or provider products.

### monarchmoney-cli

Primary reference for agent-facing CLI behavior.

Read:

- `donors/monarchmoney-cli/docs/agent-guide.md`
- `donors/monarchmoney-cli/docs/json-schema.md`
- `donors/monarchmoney-cli/docs/safety.md`
- `donors/monarchmoney-cli/internal/output/envelope.go`
- `donors/monarchmoney-cli/internal/output/render.go`
- `donors/monarchmoney-cli/internal/errors/codes.go`
- `donors/monarchmoney-cli/internal/safety/plan.go`
- `donors/monarchmoney-cli/internal/cli/root.go`
- `donors/monarchmoney-cli/internal/cli/accounts.go`
- `donors/monarchmoney-cli/internal/cli/transactions.go`

Extract these ideas:

- `--json` for machine output.
- stdout contains parseable JSON only.
- stderr contains diagnostics.
- Stable success/error envelopes.
- Stable exit codes.
- Read-only/dry-run/confirm gates for future mutations.
- Local write operations require `--dry-run` or `--confirm`; provider link/sync are explicit commands and only destructive provider operations require confirmation.
- Global read-only mode uses `MONEY_READ_ONLY=1` or `read_only: true` and blocks sync because sync writes the local store.
- Command documentation must change with command behavior.

Do not carry over:

- Monarch GraphQL as a core dependency.
- Monarch remote account/transaction field names as canonical fields without review.
- The thin cache schema as the long-term store schema.

### Actual Budget

Reference for local-first budgeting and automation shape.

Read later, after first Plaid sync and read contracts are stable:

- `donors/actual/README.md`
- `donors/actual/packages/api/README.md`
- `donors/actual/packages/api/models.ts`
- `donors/actual/packages/api/methods.ts`

Extract these ideas:

- Package separation between core logic and automation API.
- Budgeting vocabulary and account/category flows.
- Import/export ergonomics.
- Import Source idempotency should be conservative: local transaction IDs plus `import_source_id`, `import_batch_id`, and deterministic `source_row_hash`; same-batch duplicate rows can be rejected, but cross-batch/source/provider overlap is only reported as `possible_duplicate` in import results for future review.

Do not carry over:

- App-first assumptions.
- Required sync server assumptions.
- Budgeting complexity before `accounts.list` and `transactions.search` are stable.

### Maybe Finance

Reference for domain vocabulary and product modeling only.

Read later, only for domain comparison:

- `donors/maybe/README.md`
- `donors/maybe/app/models/account.rb`
- `donors/maybe/app/models/entry.rb`
- `donors/maybe/app/models/transaction.rb`
- `donors/maybe/app/models/rule.rb`
- `donors/maybe/app/models/plaid_item/syncer.rb`
- `donors/maybe/app/models/plaid_account/importer.rb`

Extract these ideas:

- Account, entry, transaction, balance, merchant, category, rule vocabulary.
- Net-worth and balance-sheet product modeling.

Do not carry over:

- Rails/PostgreSQL architecture.
- AGPL source code.
- Assistant, chat, provider OpenAI, or full web-app assumptions.

## Target Module Shape

Keep the architecture small and deep.

### `internal/contracts`

Owns Command Contracts.

Responsibilities:

- Success and error envelopes.
- Schema version.
- Stable error codes.
- Maintain one shared error code registry; do not create command-specific or module-specific error code sets.
- Error code categories follow `monarchmoney-cli`: `auth`, `network`, `api`, `validation`, `safety`, `config`, and `internal`.
- First-version exit codes follow `monarchmoney-cli`: `0` success, `1` internal or unclassified failure, `2` invalid command arguments, `3` authentication or Provider authorization required, `4` read-only violation, `5` network failure, `6` Provider/API/schema/feature failure, `7` validation failure, and `10` confirmation required.
- Add `CONFIG_WRITE_FAILED` to the shared registry for config/env file write failures; category `config`, retryable false by default, exit code `1`; message may include failed file path but never secret values or file contents.
- Add `DB_BACKUP_FAILED` to the shared registry for failure to create a required DB backup before repair/migration/re-encryption/destructive maintenance; category `safety`, retryable false by default, exit code `1`.
- Keep schema version at `0.1` during development and promote first stable read contracts to `1.0` after hardening.
- Error arrays with `code`, `message`, `category`, and `retryable`.
- Warning arrays use structured objects with `code`, `message`, and `category`.
- Pagination metadata.
- Pagination lives in `meta.pagination` with `limit`, `offset`, `total`, and `has_more` when available.
- Command metadata.
- JSON writing helpers.
- List data is object-wrapped using plural fields such as `data.transactions`, never a top-level array.
- Local stable IDs are immutable strings with explicit prefixes: `acc_`, `tx_`, `cat_`, `tag_`, `pi_`, and `inst_`.
- Local ID suffixes are cryptographically random base32/base58-style strings and do not encode provider, institution, account mask, date, merchant, or other business information.
- Provider-native IDs remain separate provenance fields such as `provider_external_item_id`, `provider_account_id`, and `provider_transaction_id`; they are not local primary IDs.
- Account records expose top-level `id` as local `acc_...`; transaction records expose top-level `id` as local `tx_...` and top-level `account_id` as local `acc_...`.
- Command filters, joins, and write targets use local top-level IDs; `source` only describes provenance.
- Account and transaction read contracts expose provenance through a uniform `source` object with `kind` values `provider`, `manual`, and `import`.
- Demo is a runtime environment, not a `source.kind`; demo fixtures still use provider, manual, or import source semantics and JSON envelopes use `meta.demo: true`.
- The `source` object includes `provider`, `provider_item_id`, `provider_external_item_id`, `institution_id`, `provider_account_id`, `provider_transaction_id`, `import_source_id`, and `import_batch_id`.
- `provider_item_id` is the local `pi_...` Provider Item ID; `provider_external_item_id` is the provider-native item identifier.
- First-version stable contracts output every `source` field and use JSON `null` for fields that do not apply.
- Provider-sourced records expose provider-native account and transaction object IDs in `source`, but never expose provider tokens, credentials, full raw payloads, full account numbers, or routing numbers.
- Date-only fields use `YYYY-MM-DD`; datetime fields use ISO 8601 UTC, following Plaid-style date/datetime separation.
- Stable contracts use `currency`; provider raw currency fields stay internal unless intentionally exposed.
- Do not persist full Provider raw payloads in the first schema; store only mapped fields and necessary provenance/status/cursor metadata.
- Account schema supports provider name/official name, optional mask, and local alias; alias is not overwritten by sync.
- Manual account balance input is unsigned, rejects plus/minus signs, normalizes thousands separators, and dry-run/confirm shows account name, signed balance, currency, and whether it increases or decreases financial position. Human final confirmation defaults to No.
- Human interactive `accounts create-manual` asks account name, account type, optional subtype, currency, unsigned balance, financial-position confirmation, and optional alias, then shows a write plan and requires final confirmation unless `--confirm` or `--dry-run` was supplied.
- Human choice prompts use Hermes Agent-style arrow-key selection with Enter to confirm, not numbered free-form menus. Apply this consistently to setup Provider selection, setup review-workflow choice, Provider credential overwrite confirmation, institution/provider choices during link, manual account type selection, and future interactive write choices. Non-TTY or automated usage must provide flags or receive validation errors for missing fields.
- JSON write commands require explicit `--dry-run` or `--confirm`; they never prompt.
- Manual transaction commands later support both flags and one-question-at-a-time interactive input for direction and unsigned amount.
- Store/core money values use integer minor units or equivalent fixed-precision decimals; JSON contracts expose monetary values as decimal strings.
- Encrypted SQLite uses `github.com/ncruces/go-sqlite3` with its `database/sql` driver and `github.com/ncruces/go-sqlite3/vfs/adiantum` encrypted VFS. Real stores must use the encrypted VFS and the configured 32-byte key. Demo uses in-memory SQLite without encryption because it never opens or writes real user data.

Rules:

- JSON mode writes only to stdout.
- JSON warnings go in the envelope's `warnings` array; human-mode diagnostics go to stderr.
- Contract tests assert shape, not implementation.

### `internal/config`

Owns configured state.

Responsibilities:

- Resolve config file path and environment variables.
- Implement the concrete config loading flow in `docs/CONFIG.md`.
- Load database path.
- Load app-managed database encryption key.
- Load provider credentials.
- Support `~/.money/config.yaml`, `~/.money/.env`, environment-variable, and setup-command configuration.
- Do not auto-load cwd `.env`; alternate config paths require `--config` or `MONEY_CONFIG`.
- Alternate config paths load same-directory `.env` by default unless config explicitly sets `env_file`.
- Resolve explicit `env:` secret references, such as a config field pointing to `PLAID_SECRET`, instead of applying a silent override chain or magic environment lookup.
- Secret YAML syntax is an object with exactly one `env` key, for example `encryption_key: {env: MONEY_DB_ENCRYPTION_KEY}` or block form with `env: MONEY_DB_ENCRYPTION_KEY`.
- Interactive setup and `money providers configure <provider>` write secrets to the resolved env file and config `env:` references, not direct YAML secret values.
- Human secret entry prompts display mask characters such as `••••` while typing.
- Confirmation, summary, JSON, diagnostics, and errors may show env variable names and boolean secret status only; they must not show raw secret values or reversible partial previews.
- Manually edited direct credential values in `config.yaml` are tolerated, but config loading and doctor return a warning recommending `env:` references for secrets.
- Existing env vars are not overwritten by default; human mode asks before replacement using the standard arrow-key selector, and JSON/non-interactive mode requires `--force`.
- If the user keeps an existing env var, update YAML `env:` references to point at it and let doctor diagnostics validate the value.
- Provider configuration is atomic per Provider; do not save partial Provider credential blocks when required fields are missing or invalid.
- Do not attempt complex cross-file rollback between `.env` and YAML. On write failure, stop immediately and report what was written, what failed, and how to rerun or repair manually without printing secrets.
- After `money providers configure <provider>` writes config, run shared doctor checks and display only that Provider's diagnostics plus global blocking diagnostics. Do not duplicate doctor rules or show unrelated Provider warnings.
- Validate command-specific requirements.
- Setup question order is a user experience decision. Engineering defaults are: validate config path first, create or generate local encryption material before opening the store, then collect optional Provider credentials, then optional review workflow settings. Setup choices use the same arrow-key selector as other human-mode choices; free-text prompts are only for values such as paths and secrets.
- Setup output summarizes written config, env, and database paths and whether secrets were created, but never prints secret values. JSON setup output uses booleans such as `secret_created: true`.
- Setup does not ask advanced path questions by default; advanced users can edit `config.yaml` or run with explicit `--config`/`MONEY_CONFIG`.
- Setup creates or opens the encrypted SQLite database and runs migrations after DB key/config are ready, giving read commands a stable encrypted empty-state.
- If setup DB open or migrations fail after config/env writes, keep written config/env, stop immediately, report the DB/migration failure and files already written, and let the user repair and rerun setup or doctor.
- If an existing DB file cannot be opened, setup must not create a replacement DB at that path.
- Future DB repair, migration, re-encryption, or destructive maintenance operations must create a same-directory UTC timestamp `.bak` backup first, e.g. `money.db.20260510T143012Z.bak`, append a short random suffix on collision, and tell the user the backup path before proceeding. If backup creation fails, stop and return `DB_BACKUP_FAILED`. Once created, keep the backup whether the DB operation succeeds or fails; on failure, report the backup path for manual recovery. Setup does not perform DB repair.
- After setup writes configuration, run the shared doctor checks against the real environment and display the returned diagnostics; do not duplicate doctor diagnostic rules in setup.
- Setup completion does not suggest demo mode; demo remains discoverable through help only.

Rules:

- Read commands require only an encrypted local database that can be opened with the configured local database key, or a documented encrypted empty-state path.
- Provider sync requires provider credentials.
- No AI keys exist in config.
- No Ray, Monarch, or donor fallback paths.
- No managed proxy keys or subscription-account config exist.
- `money` settings use `MONEY_*` names; provider credentials use provider-owned names such as `PLAID_*`.
- Provider names should follow official/common provider vocabulary: Plaid uses `PLAID_CLIENT_ID`, `PLAID_SECRET`, `PLAID_ENV`, `PLAID_PRODUCTS`, `PLAID_COUNTRY_CODES`, and `PLAID_REDIRECT_URI`; Bridge uses `BRIDGE_CLIENT_ID` and `BRIDGE_CLIENT_SECRET`; MX uses `MX_CLIENT_ID` and `MX_API_KEY`; Finicity uses `FINICITY_APP_KEY`, `FINICITY_PARTNER_ID`, and `FINICITY_PARTNER_SECRET`.

### `internal/store`

Owns encrypted SQLite persistence.

Responsibilities:

- Open local encrypted SQLite database.
- Run migrations.
- Implement the migration contract in `docs/SCHEMA.md`.
- Store institutions, provider items, accounts, transactions, and sync cursors.
- Query accounts and transactions for CLI contracts.
- Preserve provider, provider item, and provider account provenance for future merge review.
- Use provider provenance uniqueness, such as `(provider, provider_item_id, provider_record_id)`, for sync idempotency and dedupe, not readable local IDs.
- Enforce provider transaction idempotency with Provider Item plus provider-native transaction ID, such as `(provider_item_id, provider_transaction_id)`.
- Manual, demo, and imported transactions use local IDs plus explicit source metadata.
- Imported transactions record `import_source_id`, `import_batch_id`, and deterministic `source_row_hash`.
- Same-batch import duplicate rows can be rejected; cross-batch/source/provider overlap can only be reported as `possible_duplicate` in import command results.
- `possible_duplicate` is not part of the first stable `transactions.list` or `transactions.search` contracts.
- Do not implement heuristic transaction auto-merge by amount, date, merchant, or account similarity.

Initial schema:

- `institutions`
- `provider_items`
- `accounts`
- `transactions`
- `transaction_tags`
- `categories`
- `tags`
- `recurring`
- `sync_runs`

Do not create AI tables. Do not add automatic merge/dedupe tables, budgets, goals, scores, holdings, or liabilities until first account and transaction contracts are stable. A small `recurring` table is part of the first schema only because `recurring.list` is a first stable read contract.

### `internal/providers`

Owns provider seam.

Responsibilities:

- Define provider adapter interface.
- Register built-in providers.
- Expose command-specific configured-state checks.
- Return canonical records, not CLI envelopes.

Initial interface should support:

- Provider name.
- Institution discovery/search support where the Provider can offer it.
- Config validation.
- Link start/exchange flow.
- Sync all linked items.
- Sync one linked item.
- Short-lived localhost callback helpers are allowed only for active Provider authorization sessions. They are not persistent servers or local APIs.

Avoid a plugin runtime in this phase.

### `internal/providers/plaid`

Owns Plaid adapter implementation.

Responsibilities:

- Build Plaid client from explicit Plaid credentials.
- Create Link token.
- Exchange public token.
- Store encrypted access token through store interface.
- Sync accounts through Plaid Accounts API.
- Sync transactions through Plaid Transactions Sync API.
- Maintain Plaid cursor per item.

Rules:

- Use the official Go Plaid SDK.
- Keep Plaid request/response types inside the adapter.
- Map Plaid records into canonical store records.
- Convert Plaid outflow-positive transaction amounts into canonical inflow-positive/outflow-negative amounts.
- Default Plaid product is `transactions`; investments and liabilities require explicit configuration and are not stable first-milestone read contracts.
- Return explicit errors for unsupported products, missing credentials, invalid tokens, and relink-required states.
- Sync recurring transaction streams when Plaid returns them; `recurring.list` returns an empty success result when no recurring data has been synced.
- Do not call hosted Ray proxy APIs.
- Do not require AI API keys.

### `internal/providers/bridge`

Owns Bridge adapter implementation.

Responsibilities:

- Build Bridge client from explicit Bridge credentials.
- Create Bridge users and connect sessions.
- Generate and store Bridge external user IDs by default, with advanced support for existing external user IDs.
- Track Bridge item status and reconnect-required state.
- Sync Bridge accounts and transactions into canonical records.
- Maintain Bridge incremental transaction cursor/state.

Rules:

- Keep Bridge request/response types inside the adapter.
- Map Bridge records into canonical store records.
- Convert Bridge transaction amounts into canonical inflow-positive/outflow-negative amounts.
- Return explicit errors for missing credentials, authorization failures, reconnect-required states, and unsupported responses.
- Do not require AI API keys or managed proxy config.

### `internal/core`

Owns Finance Primitives.

Responsibilities:

- `ListAccounts`
- `ListTransactions`
- `SearchTransactions`
- `ListCategories`
- `ListTags`
- `ListRecurring`
- Later: budgets, cashflow, rules, merge review, net worth.

Rules:

- Core depends on store interfaces and provider interfaces.
- Core does not import Cobra, terminal rendering, Plaid SDK, or SQLite driver details.
- Core owns deterministic sorting and pagination semantics when they are part of a Command Contract.

### `internal/cli`

Owns CLI command wiring.

Responsibilities:

- Use Cobra for command routing, flags, aliases, help generation, and shell completion.
- Parse commands and flags.
- Choose human or JSON output.
- Call core services.
- Render contracts.
- Enforce stdout/stderr separation.
- Use `github.com/olekukonko/tablewriter` for human-mode tables.
- Use `charm.land/huh/v2` behind an internal prompt interface for arrow-key selectors and human-mode forms.

Rules:

- Use Go's standard `testing` package by default. Add `testify` only if repeated assertion boilerplate obscures test intent.
- Do not add Viper; config loading is explicit and owned by `internal/config`.
- Keep terminal UI and table rendering out of core.

Initial commands:

- `money version`
- `money setup`
- `money accounts list --json`
- Later write: `money accounts update <id> --alias <name> --dry-run|--confirm --json`
- `money accounts create-manual --dry-run|--confirm --json`
- `money demo <command...>`
- Main help lists `demo` with low-key text such as `Run a command against bundled non-persistent sample data`; setup completion and doctor remediation do not recommend demo.
- `money transactions list --json`
- `money transactions search <query> --json`
- `money tx list --json` alias
- `money tx search <query> --json` alias
- `money categories list --json`
- `money tags list --json`
- `money recurring list --json`
- `money link`
- `money providers configure plaid`
- `money providers configure bridge`
- `money providers plaid link`
- `money providers bridge link`
- `money sync`
- `money sync --provider <provider>`
- `money sync --provider-item <id>`
- `money doctor`
- `money doctor --json` provisional diagnostic contract
- Later: `money transactions cleanup --removed --dry-run|--confirm --json`

Only `accounts.list`, `transactions.list`, `transactions.search`, `categories.list`, `tags.list`, and `recurring.list` are stable contracts in the first milestone.

Provider sync, demo mode, and manual account creation are first-version capabilities. Monarch/CSV/Apple Card imports and transaction annotation write commands are deferred unless needed to validate the first stable read contracts.

## Phased Work Plan

### Phase 1: Contract and Store Foundation

Goal: stable local read contracts against encrypted SQLite without any live Provider calls.

Steps:

1. Add `internal/config`.
2. Add encrypted SQLite dependency.
3. Implement store open/close and migrations.
4. Create minimal tables: institutions, provider_items, accounts, transactions, transaction_tags, categories, tags, recurring, sync_runs.
5. Support manual accounts with nullable Provider Item provenance and local source metadata.
6. Support a non-persistent demo store seeded with small, inspectable synthetic data for local testing and trial use without Provider credentials.
7. Implement store methods for listing accounts, creating manual accounts, listing/searching transactions, listing categories, listing tags, and listing recurring items against both real and demo stores.
8. Add fixture seed helpers for tests.
9. Implement CLI skeleton for `accounts list`, `accounts create-manual`, `demo <command...>`, `transactions list`, `transactions search`, `tx` aliases, `categories list`, `tags list`, and `recurring list`.
10. Add JSON contract tests for all first-milestone read commands, manual account dry-run/confirm behavior, and demo isolation/reset behavior.
11. Run `go test ./...`.

Acceptance:

- `money accounts list --json` returns a valid envelope from an empty or seeded local DB.
- `money accounts list --json` shows all unmerged accounts and includes provider provenance.
- Human `accounts list` defaults to `NAME`, `TYPE`, `BALANCE`, `AVAILABLE`, `CURRENCY`, `SOURCE`, and `UPDATED`; `--verbose` may show provider raw IDs and deeper provenance.
- `money accounts create-manual --dry-run --json` returns a plan without writing.
- `money accounts create-manual --confirm --json` creates a local manual account.
- `money demo <command...>` runs the command against synthetic in-memory data without reading or writing the real encrypted store.
- Demo writes are allowed inside the demo runtime and reset when the demo run ends.
- Demo mode only swaps the store; command logic, safety gates, read-only behavior, contracts, and validation are reused unchanged.
- Demo output clearly indicates demo mode with `meta.demo: true` in JSON and this human-mode banner: `Demo mode: using bundled non-persistent sample data. Changes are discarded when this command exits.`
- Demo mode must not call real Providers; demo link/sync uses synthetic fixtures or demo-only adapters.
- Demo mode does not require DB encryption key or open the real store, but may read non-secret runtime settings such as read-only mode.
- Demo data is a small deterministic mock database bundled into the program and never writes user config or the real database.
- Demo fixtures include a few examples for every first-version feature: multiple account types, provider/manual/import provenance, categories, tags, notes, recurring items, pending transactions, removed transactions, and review-state examples.
- Demo reuses main store, migrations, queries, commands, contracts, validation, and safety logic; prefer in-memory SQLite plus fixtures over a fake store.
- `money transactions list --json` returns deterministic recent or filtered results.
- `money transactions search <query> --json` returns deterministic results.
- `money transactions search <query> --json` includes pending transactions by default and exposes pending state.
- Human transaction list/search defaults to `DATE`, `ACCOUNT`, `MERCHANT`, `AMOUNT`, `CATEGORY`, and `STATUS`; `STATUS` shows compact flags such as `pending`, `review`, and `removed`.
- Transaction `--verbose` may show local IDs, source provenance, note, tags, provider category, and other diagnostic fields.
- Human money output includes explicit text signs; color is auxiliary only. Use US stock-market convention: green for positive/inflow/up values and red for negative/outflow/down values.
- Transaction list/search supports filters for category, account, merchant, tag, needs-review, pending, and recurring status when corresponding fields exist.
- Split transaction state is deferred; Ray Finance does not model split state in its transaction schema/query projection, and Monarch split commands require a separate local design before adoption.
- Transaction list/search supports `--removed exclude|include|only`, defaulting to `exclude`.
- Tags, notes, review state, and custom transaction fields are local annotations, not Provider-supplied data.
- Transaction read contracts expose both `tag_ids` and readable `tags` objects with at least `id` and `name`; both represent the same local annotations.
- Transaction read contracts expose a single local `note` string or `null`; rich text, comment threads, and attachments are out of scope for the first contract.
- Provider sync preserves local annotations and only updates provider-owned fields unless an explicit user override model exists.
- Category uses provider category/subcategory plus local category override fields.
- Transaction read contracts expose top-level `category_id`, `category_name`, and `category_source`; provider raw classification stays in `provider_category` and `provider_subcategory`.
- `category_source` values are `local`, `provider`, `import`, and `none`.
- Provider sync does not create local categories from provider category names; provider-only classification keeps `category_id: null`.
- Review workflow is local, optional, configured during setup with the selector question `Mark newly synced transactions as needing review?`, and disabled by default with No selected.
- Enabling review during setup marks only newly synced transactions as `needs_review=true`; existing transactions remain unchanged.
- First-version transaction contracts expose a boolean `needs_review`; enums, assignees, workflow history, and review comments are out of scope.
- Setup may skip Provider credentials by default, but must warn that sync/link are unavailable and show help-style guidance for adding credentials later.
- Skipping Provider credentials leaves local reads, demo mode, and manual data available.
- Provider configure guidance is generated from the command registry and registered Provider list, not hard-coded provider command strings.
- `money doctor --json` reports config, encrypted store, Provider credential, linked item, sync readiness, and warning state, but is provisional until diagnostic schema hardening.
- Human `money doctor` groups checks under `Config`, `Store`, `Providers`, `Links`, `Sync`, and `Warnings`, with `ok`, `warn`, and `error` status labels.
- Overly broad `.env` permissions are `warn` when they do not block use; tell the user `doctor --fix` can repair them. On Unix-like platforms, repair sets the secrets file to `0600`; platforms without POSIX chmod semantics warn instead.
- First-version permission checks focus on the secrets env file, not `config.yaml`; direct YAML secrets are covered by the direct-secret warning.
- `money doctor` is read-only by default; `money doctor --fix` is explicit repair mode for supported configuration issues.
- Human `money doctor --fix` executes supported non-destructive repairs directly; `--dry-run` may show the repair plan without writing.
- `doctor --fix` must be idempotent and must not overwrite user-set values, remove unknown fields, rewrite credentials, change existing configured paths, migrate donor config, move or re-encrypt databases, purge data, link accounts, or sync.
- First `doctor --fix` scope: create local config directory, config skeleton, `.env` skeleton, missing DB encryption key, and default DB path only.
- Generated DB encryption keys use at least 256 bits of cryptographic randomness, stored as `MONEY_DB_ENCRYPTION_KEY`, and are never printed in full JSON output.
- `tx` aliases return the same `meta.command` values as canonical `transactions` commands.
- `money categories list --json`, `money tags list --json`, and `money recurring list --json` return deterministic local read results.
- Transaction search default order is `date DESC, pending DESC, id ASC`.
- Read commands do not require provider credentials.
- Read commands fail explicitly when the database key is missing or cannot decrypt the store.
- Accounts and transactions retain provider item provenance.
- Tests prove stdout contains JSON only in JSON mode.

### Phase 2: Plaid and Bridge Config and Provider Adapters

Goal: prepare Plaid and Bridge as replaceable Provider adapters.

Steps:

1. Add official Go Plaid SDK.
2. Add Bridge HTTP client implementation.
3. Add Plaid config fields with `PLAID_` environment variable names.
4. Add Bridge config fields with `BRIDGE_` environment variable names.
5. Add provider registry with Plaid and Bridge adapters.
6. Define canonical provider records for Institution, ProviderItem, FinancialAccount, Transaction, and SyncResult.
7. Write fixture-based mapper tests before any live Provider calls, including canonical amount sign normalization and liability balance normalization.
8. Implement Plaid client construction without managed proxy behavior.
9. Implement Bridge client construction without managed proxy behavior.
10. Implement provider error classification.

Acceptance:

- Missing Plaid credentials produce a stable config error for Plaid commands.
- Missing Bridge credentials produce a stable config error for Bridge commands.
- Read commands still work without Plaid credentials.
- Read commands still work without Bridge credentials.
- Plaid and Bridge mapper tests cover account and transaction fixture payloads.
- No code references `RAY_API_KEY`, `RAY_PROXY_BASE`, `rayfinance.app`, Anthropic, OpenAI, or chat.
- No code requires a `money` account, subscription, managed proxy key, or AI API key.

### Phase 3: Plaid Link Flow

Goal: allow the user to create a Plaid item and store its access token inside the encrypted store.

Steps:

1. Implement `money link` for institution-first linking through Plaid when Plaid is the selected Provider.
2. Implement `money providers plaid link` for explicit Plaid linking.
3. Separate Provider support from Provider availability: list all Providers known to support the selected institution, mark locally unavailable Providers as missing credentials, and block selecting unavailable Providers with generated configure guidance.
4. Determine Provider availability from local config/env required fields, following Ray's configured-state pattern but using `money`'s explicit `env:` model.
5. Use Plaid `/institutions/search` with configured products and country codes for Plaid institution discovery.
6. Use a short-lived localhost Link page and callback server only for Plaid Link.
7. Create Plaid Link token with explicit products and country codes.
8. Serve a local page that loads Plaid Link with the link token and posts `public_token`, selected institution/account metadata, and random state back to the localhost helper on success.
9. Exchange public token for access token.
10. Store access token only after the encrypted store is open.
11. Store provider item, institution metadata, products, and initial cursor state.
12. Use random state/nonce values, timeout callback sessions, bind only for the active link flow, and shut down the helper after completion.
13. Follow GitHub CLI browser ergonomics in human mode: print the local Link URL, wait for Enter, then open the browser.
14. Support `--no-open` to print the URL without opening a browser for SSH, cron, and headless environments.
15. Do not automatically run the first sync after link; print help-derived guidance for running `money sync`.
16. Add tests for encrypted-store requirements and link exchange logic using fakes.

Acceptance:

- Link flow never starts a general Local API server.
- Plaid Link uses a localhost page backed by a Plaid link token; it does not print a fake raw Plaid authorization URL.
- Human link flow waits for explicit Enter before opening the browser.
- If the user does not press Enter, no browser opens and the printed URL remains available for manual/headless handling.
- `--no-open` does not attempt to open a browser.
- Access tokens are not stored outside the encrypted store.
- Link stores the Provider Item but does not fetch account or transaction data through sync unless the user runs `money sync`.
- The command fails explicitly if the database key is missing.
- The command does not require AI or hosted service config.

### Phase 3B: Bridge Link Flow

Goal: allow the user to create or reconnect Bridge items.

Steps:

1. Extend `money link` to choose Bridge when Bridge supports the selected Institution.
2. Implement `money providers bridge link` for explicit Bridge linking.
3. Use the same supported-versus-available Provider selection rules as Plaid.
4. Create or reuse a Bridge external user ID.
5. Create a Bridge connect session.
6. Print the connect session URL, wait for Enter, then open the browser unless `--no-open` is set.
7. Poll Bridge items until a new or reconnected item is available.
8. Store provider item, institution metadata, status, and cursor state in the encrypted store.
9. Do not automatically run the first sync after link; print help-derived guidance for running `money sync`.
10. Add tests for Bridge connect/session/state logic using fakes.
11. If Bridge institution discovery cannot be implemented cleanly in the first adapter, keep `money providers bridge link` working and defer institution-first Bridge selection rather than inventing unsupported discovery.

Acceptance:

- Bridge link flow stores item state in the encrypted store.
- Reconnect-required states are explicit.
- The command fails explicitly if Bridge credentials are missing.
- The command does not require AI or hosted service config.

### Phase 4: Provider Account and Transaction Sync

Goal: sync Plaid and Bridge accounts and transactions into canonical encrypted SQLite tables.

Steps:

1. Implement `money sync`.
2. Load linked provider items from the store.
3. Load provider access tokens from the encrypted store per item.
4. Sync accounts first.
5. Sync transactions using Plaid Transactions Sync cursors.
6. Sync Bridge transactions using Bridge updated-at cursor/state.
7. Upsert provider transactions by Provider Item plus provider-native transaction ID.
8. Handle added, modified, and removed transactions.
9. Store next cursor only after successful transaction processing.
10. Record a sync run with counts and status.
11. Add fixture-backed sync tests with fake Plaid client.

Acceptance:

- Sync is idempotent.
- `money sync` syncs all linked Provider Items by default and supports provider/provider-item narrowing.
- No linked Provider Items returns a successful empty sync result with a warning, not an error, so local-database-only users are supported.
- Sync returns a structured summary with per-provider-item status and partial results.
- Human sync output defaults to compact summary; `--verbose` shows per-Provider-Item status.
- JSON sync output always includes per-Provider-Item status.
- Any provider item failure makes JSON `ok=false` while preserving partial result data for diagnosis.
- Sync JSON is provisional until provider error taxonomy, reconnect handling, removed transaction policy, and partial failure behavior are tested and documented.
- Modified transactions update existing rows.
- Removed transactions are marked or deleted according to the documented schema decision.
- Removed transactions are soft-deleted by sync, hidden from default reads, and purgeable only through an explicit dry-run/confirm cleanup command.
- Cursors advance only after successful processing.
- Partial provider failures are reported explicitly.
- `money accounts list --json`, `money transactions list --json`, `money transactions search <query> --json`, `money categories list --json`, `money tags list --json`, and `money recurring list --json` read synced data without Plaid credentials.

### Phase 5: Contract Hardening

Goal: make the first-milestone read Command Contracts reliable enough for external agents.

Steps:

1. Document `accounts.list`, `transactions.list`, `transactions.search`, `categories.list`, `tags.list`, and `recurring.list` schemas.
2. Add pagination metadata.
3. Add stable error codes.
4. Add deterministic ordering:
   - Accounts: hidden flag, type, name, id.
   - Transactions: date descending, pending status, id.
5. Add filters for account, category, merchant, tag, date range, needs-review, pending, recurring status, and limit.
6. Add command examples.
7. Add compatibility notes for Monarch migration.

### Phase 6: Sync Contract Hardening

Goal: promote sync JSON from provisional to stable after real Provider behavior is understood.

Steps:

1. Document sync summary schema.
2. Define provider error taxonomy.
3. Define reconnect-required behavior.
4. Define removed transaction policy as soft delete plus explicit confirmed cleanup.
5. Define partial failure semantics and exit codes.
6. Add fixture tests for Plaid and Bridge partial failures.
7. Add command examples for cron usage.

Acceptance:

- `money sync --json` has a documented schema.
- Partial failures return `ok=false` with per-item diagnostics.
- Reconnect-required states are machine-readable.
- Removed transaction behavior is stable and tested.
- Provider errors are classified into stable categories for missing credentials/config, invalid authorization, reconnect required, rate limit, network failure, unsupported product/feature, provider schema/API change, validation failure, and internal failure.

Acceptance:

- Contracts are documented.
- Contract tests cover success, empty state, validation error, and database error.
- Agents can parse errors without reading stderr.

### Phase 7: Next Provider and Migration Work

Goal: prove the Provider seam is real with a second adapter.

Choose one:

- Monarch import from `monarchmoney-cli --json`.
- CSV import.
- MX or Finicity provider adapter.

Acceptance:

- A second adapter can write the same canonical tables.
- Existing account and transaction contracts do not change.
- Source/provider-specific fields remain namespaced or internal.

## Initial Schema Direction

Keep the first schema small. Do not copy Ray's full schema.

Use canonical naming:

- `institutions`: user-visible financial institutions or manual sources.
- `provider_items`: provider linkage state, encrypted token, products, cursor, reconnect/status state.
- `accounts`: canonical Financial Accounts.
- `transactions`: canonical Transactions.
- `transaction_tags`: local transaction/tag relation.
- `categories`: local category definitions.
- `tags`: local tag definitions.
- `recurring`: lightweight recurring streams/items available from local provider data or fixtures.
- `sync_runs`: sync audit trail.

Keep these for later:

- holdings
- securities
- liabilities
- budgets
- goals
- rules
- net_worth_history
- daily_scores

The concrete first DDL is maintained in `docs/SCHEMA.md`; update that file before changing migration shape.

Never add these:

- conversation_history
- memories
- ai_audit_log
- model provider tables
- hosted billing tables

## Provider Mapping Rules

Plaid maps into canonical records:

- Plaid item -> provider item.
- Plaid institution -> institution.
- Plaid account -> account.
- Plaid transaction -> transaction.
- Plaid account ID remains provider account ID and can also be used as initial account ID.
- Plaid transaction ID remains provider transaction ID and can also be used as initial transaction ID.

Canonical records must include:

- source/provider.
- provider item ID.
- provider record ID.
- account ID.
- timestamps.
- currency.
- pending status for transactions.

Avoid leaking Plaid-specific product fields into `accounts.list` or `transactions.search` unless namespaced under provider metadata and documented.

## Error Handling Rules

Use explicit failures:

- Missing database path.
- Database cannot open.
- Migration failed.
- Missing provider credentials for provider commands.
- Missing database encryption key for store-backed commands.
- Plaid item requires relink.
- Plaid product unavailable.
- Plaid rate limit or network failure.

Do not implement hidden fallback behavior:

- No fallback from `money` config to Ray config.
- No fallback from Plaid BYOK to Ray managed proxy.
- No fallback from encrypted database failure to plaintext SQLite.
- No fallback from missing provider credentials to demo data.
- Partial sync or multi-error commands return the most severe applicable exit code, not a special fallback code.

## Verification Commands

Run these after every implementation phase:

```bash
go test ./...
go run ./cmd/money version
git status --short --ignored
```

Before merging provider work, also run:

```bash
rg -n "RAY_API_KEY|RAY_PROXY_BASE|rayfinance\\.app|Anthropic|OpenAI|conversation_history|ai_audit_log|memories" .
```

The search may match donor files under `donors/`; ignore those matches only if all runtime code is clean. Runtime code means everything outside `donors/`.

## Done Definition

Provider support is considered initially complete when:

- Plaid credentials can be configured explicitly.
- Bridge credentials can be configured explicitly.
- Plaid Link stores access tokens only inside the encrypted store.
- Bridge link stores provider item state only inside the encrypted store.
- Plaid and Bridge sync write accounts and transactions into encrypted SQLite.
- `money accounts list --json` returns synced accounts.
- `money transactions list --json` returns synced transactions.
- `money transactions search <query> --json` searches synced transactions.
- `money categories list --json`, `money tags list --json`, and `money recurring list --json` return synced local read data when available.
- Read commands work without provider credentials after sync.
- Stable contract tests pass.
- No runtime code imports or references donor AI, hosted Ray, billing, or chat concepts.
- README and donor acknowledgements remain accurate.
