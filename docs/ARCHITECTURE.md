# Architecture

## Shape

`money` is organized around four modules:

- `contracts`: JSON envelopes, schema versions, and command contract metadata.
- `core`: canonical finance types and deterministic finance primitives.
- `store`: encrypted SQLite persistence and query implementations.
- `providers`: networked financial data adapters such as Plaid, Bridge, MX, and Finicity.

The CLI calls core services. Core services call store interfaces and provider interfaces. Providers do not own command output. The store does not know about CLI rendering. Command Contracts are the stable interface for agents, scripts, and cron jobs.

## Reference Priority

When a behavior can be answered from donors, prefer existing donor evidence over inventing a new answer. Ray Finance is the primary engineering donor for local data model decisions, Provider sync, Plaid/Bridge flow, encrypted storage behavior, imports, local annotations, and query fields. `monarchmoney-cli` is the primary reference for command naming habits, JSON envelope shape, stdout/stderr separation, exit codes, safety gates, and agent-facing ergonomics. If Ray Finance lacks a feature, do not invent a first-version stable contract from Monarch alone; record the feature as deferred unless it is required by `money`'s goals. If either donor conflicts with `money` boundaries, the `money` boundary wins and the decision should be made explicitly.

Engineering decisions default to current best engineering and security practices, then donor evidence, then project constraints. Do not ask the user to choose implementation details unless the decision changes user experience, product boundaries, risk acceptance, security posture, or long-term interoperability. User-facing behavior, naming, prompts, and workflow trade-offs remain product decisions.

## Data Flow

```text
Provider service -> provider adapter -> canonical records -> encrypted SQLite store -> core query -> JSON contract -> external agent
```

Read commands should use local data only. Sync commands are the explicit place where outbound provider calls happen.

`money sync` refreshes all linked Provider Items by default, which makes it suitable for cron jobs. `--provider <name>` narrows sync to one Provider, and `--provider-item <id>` narrows sync to one linked item. Missing credentials must be reported explicitly. No linked Provider Items is a successful empty state with a warning, not an error, because some users may use `money` only for local database management without any Provider links. Human mode exits successfully and prints a warning; JSON mode returns `ok: true` with a structured warning.

Sync output is a structured contract, not just a human status line. Human `money sync` defaults to a compact summary of provider items synced, failed count when nonzero, accounts seen/updated, transactions added/modified/removed, and warning count. `--verbose` shows each Provider Item's status. JSON output always includes per-item status and partial result data for diagnosis. If any Provider Item fails, JSON output should return `ok=false` while still including partial result data.

Sync JSON is provisional during the Plaid and Bridge milestone. The first stable contracts are read contracts; sync becomes stable only after provider error taxonomy, reconnect handling, removed transaction policy, and partial failure behavior are tested.

Removed provider transactions are soft-deleted. Sync marks them as removed and preserves the local row and annotations. Transaction read commands exclude removed records by default and support `--removed exclude|include|only`. Permanent cleanup is a separate write operation that requires dry-run and confirmation.

[Planned] The cleanup command follows Monarch-compatible transaction naming: `money transactions cleanup --removed --dry-run --json` previews permanent deletion, and `money transactions cleanup --removed --confirm --json` applies it. `tx cleanup` may exist as an alias, but cleanup must always name the cleanup target explicitly.

Write operations use Monarch-style safety gates. Local mutations such as cleanup, annotations, category/tag mutations, merge apply, and manual account changes require `--dry-run` or `--confirm`. Read commands do not. Provider link and sync are explicit outbound commands and do not require confirmation by default, but destructive relink, remove, or purge operations do.

Global read-only mode is supported through `MONEY_READ_ONLY=1` or config `read_only: true`. In read-only mode, local write operations are blocked even with `--confirm`, destructive provider operations are blocked, `doctor --fix` is blocked, and sync is blocked because it writes the local store. Read commands and read-only `doctor` remain available. `doctor --fix` may still create the initial local config directory and config skeleton when no readable config exists and read-only mode has not been enabled by a loaded config or `MONEY_READ_ONLY=1`.

Read-only resolution is safety-monotonic: effective read-only is true when either config `read_only: true` or `MONEY_READ_ONLY=1` is set. Environment variables may force read-only on but must not force it off when config enables it.

Dry-run write plans are allowed in read-only mode because they do not mutate state. Confirmed writes remain blocked.

## Output Boundary

JSON mode writes exactly one parseable envelope to stdout. Non-fatal diagnostics, such as direct secrets in `config.yaml`, belong in the envelope's `warnings` array as structured warning objects with `code`, `message`, and `category`. Human mode may print warnings to stderr. Missing required config remains an error, not a warning.

Error envelopes use `errors: []` rather than Monarch's single `error` object because `money` needs to represent partial sync and multi-diagnostic failures. Each error item should include Monarch-style fields: `code`, `message`, `category`, and `retryable`.

Error codes and process exit codes follow `monarchmoney-cli` unless a `money` boundary explicitly requires a new code. First-version categories are `auth`, `network`, `api`, `validation`, `safety`, `config`, and `internal`. First-version exit codes are: `0` success, `1` internal or unclassified failure, `2` invalid command arguments, `3` authentication or provider authorization required, `4` read-only violation, `5` network failure, `6` provider/API/schema/feature failure, `7` validation failure, and `10` confirmation required. Multi-error envelopes should use the most severe applicable exit code; partial sync still returns a non-zero exit code when any Provider Item fails.

`money` owns one shared error code registry. Do not create command-specific or module-specific error code sets. Add `CONFIG_WRITE_FAILED` to the shared registry for config/env file write failures; it uses category `config`, is not retryable by default, and maps to exit code `1` unless a more specific safety or validation code applies. Its message may include the file path that failed to write, but must never include secret values or file contents. Add `DB_BACKUP_FAILED` to the shared registry for failure to create the required pre-repair DB backup; it uses category `safety`, is not retryable by default, and maps to exit code `1`.

Pagination metadata belongs in `meta.pagination`, not command data. Paginated contracts should include `limit`, `offset`, `total`, and `has_more` when total is available.

Schema versions use semantic version strings. Breaking contract changes increment the major version, additive compatible fields increment the minor version, and implementation-only changes do not change schema version.

List contract data is always object-wrapped rather than a top-level array. Use plural collection fields: `data.accounts`, `data.transactions`, `data.categories`, `data.tags`, and `data.recurring`.

Local stable IDs are immutable strings with explicit type prefixes. First-version prefixes are `acc_` for Financial Accounts, `tx_` for transactions, `cat_` for categories, `tag_` for tags, `pi_` for Provider Items, and `inst_` for institutions. IDs use cryptographically random suffixes encoded in a URL-safe alphabet such as base32 or base58; they do not encode provider, institution, account mask, dates, merchant names, or other business information. Provider-native IDs are stored separately as provenance fields such as `provider_external_item_id`, `provider_account_id`, or `provider_transaction_id`; they are not the primary local IDs exposed in command arguments or stable contracts.

Top-level local relationship fields remain separate from provenance. Account records expose top-level `id` as the local `acc_...` ID. Transaction records expose top-level `id` as the local `tx_...` ID and top-level `account_id` as the local `acc_...` account reference. Command filters, joins, and write targets use these local IDs. The `source` object explains where the record came from; it does not replace local relationship fields.

Date and datetime fields follow Plaid-style formatting. Date-only fields, such as transaction date and authorized date, use ISO dates (`YYYY-MM-DD`). Datetime fields, such as generated, synced, updated, removed, or provider datetime fields, use ISO 8601 date-time strings (`YYYY-MM-DDTHH:mm:ssZ`) normalized to UTC when `money` creates them. Provider-supplied date-only transaction dates must remain date-only and must not be converted through local time zones.

Transactions distinguish posted/effective `date` from optional `authorized_date`. Provider datetime fields may be exposed as optional `datetime` and `authorized_datetime` only when the Provider supplies real datetime values. Sorting and date filters use canonical `date`.

Stable contracts use `currency` for ISO 4217 currency codes when known. Provider raw fields such as Plaid `iso_currency_code` remain adapter-internal unless intentionally exposed under a namespaced/raw field. Unsupported or non-ISO currency must produce an explicit mapping warning or error rather than silently defaulting to USD.

The first store schema does not persist full Provider raw payloads. Store only necessary provenance, mapped canonical fields, cursor/status state, and limited metadata required for reconnect or diagnostics. Full JSON response blobs, unnecessary identity data, and account/routing-like details should not be stored unless a later explicit decision adds them.

Accounts may store and expose provider-supplied `mask` when available to help distinguish accounts, but must not store full account or routing numbers. Users can set a local account alias; aliases are local annotations and must not be overwritten by Provider sync. Account contracts may expose provider name, official name, mask, and alias separately, with display naming derived intentionally.

Account `display_name` is derived as alias first, then provider name, then official name. Provider `name`, `official_name`, local `alias`, and `mask` remain separate fields; changing an alias must not mutate Provider-supplied names.

[Planned] The first account write operation may be `money accounts update <id> --alias <name>`, following Monarch-compatible naming. It only changes the local alias and requires dry-run or confirmation.

## Store Boundary

The store opens an encrypted SQLite database and runs migrations against that encrypted database. The encryption model is app-managed local encryption: `money` reads a configured local database key and uses it to open the store automatically.

There is no plaintext fallback for real financial data. This is not a zero-knowledge master-password vault: there is no required unlock prompt, lock flow, hosted account, subscription, or AI key. Provider credentials live inside the encrypted store; if additional field-level encryption is added later, it must not weaken the whole-database encryption requirement.

If a configured DB file already exists but cannot be opened, commands must report the failure and must not create a replacement DB, rename the existing DB, or silently switch paths. Any future DB repair, migration, re-encryption, or destructive maintenance operation must first create a `.bak` backup of the DB file and tell the user the backup path before proceeding. Backup files live next to the DB and use a UTC timestamp suffix such as `money.db.20260510T143012Z.bak`; if the path already exists, append a short random suffix. If backup creation fails, stop immediately and do not perform the DB operation. Once a backup is created, keep it whether the DB operation succeeds or fails; if the operation fails, report the backup path for manual recovery. Setup does not perform DB repair.

The concrete encrypted SQLite implementation is an engineering decision. Select the approach through a short spike using these criteria: single-binary feasibility on the target platforms, active maintenance, predictable Go driver behavior, compatibility with SQLite migrations and in-memory demo tests, no plaintext fallback, and clear handling of key material. If no mature single-binary encrypted SQLite option satisfies those constraints, document the trade-off before accepting an external native dependency.

The first implementation uses `github.com/ncruces/go-sqlite3` with its `database/sql` driver and the `github.com/ncruces/go-sqlite3/vfs/adiantum` encrypted VFS. This is the selected MVP path because it is cgo-free, supports app-managed 32-byte key material, keeps the CLI close to a single-binary distribution, works with normal SQLite migrations, and supports the in-memory demo path through the same SQLite implementation. Open encrypted databases with the encrypted VFS, pass the configured 32-byte key as the VFS `hexkey`, and set `temp_store(memory)` on the encrypted connection so temporary files do not become an accidental plaintext surface.

This choice provides encryption at rest, not tamper-proof database authentication. The first version treats local tamper detection and signed backups as later hardening work, not a reason to add a custom encryption layer or ship plaintext fallback behavior. If the encrypted VFS fails the implementation spike on the target platforms, the next option is a SQLCipher-backed driver with the native dependency documented explicitly before adoption; do not replace whole-database encryption with ad hoc field encryption.

## Provider Boundary

Provider support is BYOK-only. `money` talks directly to user-configured financial Providers such as Plaid and Bridge first, with MX and Finicity later. Providers are peers: users link an institution first, then choose a Provider when more than one Provider supports that institution. `money` must not depend on a managed provider proxy, hosted billing, subscription account, or AI API key.

Import Sources are separate from Providers. CSV, Apple Card export, and Monarch export can bring data into `money`, but they are file or migration inputs rather than live networked sync adapters.

Import Source transaction identity is conservative. Imported transactions receive local transaction IDs and record `import_source_id`, `import_batch_id`, and a deterministic `source_row_hash`. A single import batch can use `source_row_hash` to prevent duplicate rows from the same file, but cross-batch, cross-source, or import-versus-provider overlap must not auto-merge or delete records. At most, the importer may report `possible_duplicate` metadata in the import command result for a future user-reviewed reconcile flow; `possible_duplicate` is not part of the first stable `transactions.list` or `transactions.search` contracts.

Manual accounts are local Financial Accounts, not Providers and not Import Sources. Manual account support is part of the first implementation so `money` can work offline and support demo/test data without a Provider. Commands should follow Monarch-compatible naming such as `accounts create-manual` and use dry-run/confirm gates.

Manual account creation requires name, type, balance, and currency. Optional fields include subtype, institution name, as-of date, and alias. Account types include depository, credit, loan, investment, property, vehicle, other_asset, and other_liability.

In human interactive mode, `accounts create-manual` asks one question at a time in this order: account name, account type, optional subtype, currency, unsigned balance, confirmation of whether the account increases or decreases financial position, and optional alias. Choice prompts should follow the Hermes Agent setup style: use an arrow-key selector with Enter to confirm, not numbered free-form menus. Non-TTY or automated usage should provide flags; if required interactive fields are missing without a usable TTY, return a validation error. When no `--dry-run` or `--confirm` flag is provided in human mode, the command may collect fields interactively, then must display a write plan and ask for final confirmation before writing. `--confirm` skips the final confirmation prompt after validation, and `--dry-run` displays the plan without writing. JSON mode and fully flagged mode do not prompt; JSON write commands require explicit `--dry-run` or `--confirm`, and missing required fields return validation errors.

Manual account balance input is an unsigned amount. The CLI should reject explicit plus/minus signs, may normalize thousands separators such as `1,234.56`, and derives the canonical signed balance from account type. Dry-run/confirm output must clearly state the account name, signed balance, currency, and whether the account increases or decreases financial position and by what amount before saving. Human final confirmation uses the standard arrow-key selector and defaults to No.

Future manual transaction create/update commands should use unsigned amount plus explicit direction (`inflow` or `outflow`) rather than signed amount input. They may accept flags such as `--amount 25.50 --direction outflow` or run interactively, asking one question at a time for direction and amount. The stored canonical amount remains positive for inflow and negative for outflow.

Money values must use precise representations, not floating point. The store and core should use integer minor units or an equivalent fixed-precision decimal representation with currency. JSON contracts expose monetary values as decimal strings under normal field names such as `amount` and `current_balance`, not as floating point numbers.

Decimal string scale follows ISO 4217 minor units for the currency. Do not invent project-specific money formats. Unknown or non-ISO currencies require an explicit mapping decision or return a mapping error/warning instead of guessing.

Percentage inputs and fields must avoid fraction/percent ambiguity. CLI prompts and help text should state examples such as `22.24` for 22.24%, and contract fields should use explicit names such as `apr_percent` with decimal string values.

Interactive prompts are human-mode only. Write commands with missing required arguments may ask one question at a time in human mode. JSON mode must never prompt; missing arguments return a validation error envelope.

All human-mode choice prompts use the same Hermes Agent-style selector: arrow keys move between options and Enter confirms. This applies to setup Provider selection, setup review-workflow choice, Provider credential overwrite confirmation, institution/provider choices during link, manual account type selection, and future interactive write choices. Free-text input is reserved for values such as names, paths, secrets, amounts, and search text. Non-TTY or automated usage must provide flags or return validation errors for missing choices.

Human output is the default CLI mode and `--json` is opt-in, matching Monarch CLI habits. JSON contracts remain the automation source of truth and are the focus of contract tests.

JSON output is compact by default for machine use. [Planned] `--pretty` may format JSON for humans, but stdout must still contain only JSON.

The first implementation includes a demo environment so users and tests can exercise `money` without real financial data or Provider credentials. Demo mode is isolated from the real encrypted store, uses synthetic initial data, allows normal read and write operations inside the demo runtime, and resets to the initial state when the demo run ends.

The first demo entrypoint is command prefix mode: `money demo <command...>`, such as `money demo accounts list --json` or `money demo accounts create-manual ... --confirm --json`. It runs the command against an in-memory synthetic store and discards changes after the command exits. Demo mode only swaps the store; command logic, safety gates, read-only behavior, contracts, and validation must be reused unchanged so demo mode can test real behavior. Demo output must clearly tell the user it is running in demo mode: JSON envelopes include `meta.demo: true`, and human output displays a short banner before command output: `Demo mode: using bundled non-persistent sample data. Changes are discarded when this command exits.` A later interactive demo shell can be added if needed.

Demo mode must not call real Providers. Demo link/sync behavior uses synthetic fixtures or explicit demo-only adapters. Provider-specific demo commands must not open Plaid, Bridge, MX, or Finicity network flows.

Demo mode does not require the real database encryption key, does not open the real encrypted store, and does not run real migrations. It may read non-secret runtime settings such as read-only mode so normal command logic can be tested.

Demo data is a small deterministic mock database bundled into the program. Keep it simple enough to inspect by eye, but include at least a few examples for each first-version feature: multiple account types, provider/manual/import provenance, categories, tags, notes, recurring items, pending transactions, removed transactions, and review-state examples. It must not write to or depend on user configuration or the user's real database.

Demo should reuse the main store, migrations, queries, commands, contracts, validation, and safety logic. Use in-memory SQLite loaded with bundled fixtures over a separate fake store. Demo-specific code should be limited to selecting the in-memory store, seeding deterministic data, marking output as demo, and preventing real Provider calls. Demo in-memory SQLite is not encrypted and must not accept a file path; this is allowed only because it never stores or opens real user data.

`money doctor` checks the real environment. `money demo doctor --json` checks the demo runtime and returns `meta.demo: true`. Real doctor may report that demo mode is available, but demo readiness must not be treated as real configuration health.

The canonical linking command is `money link`: it starts from an institution search/selection and then asks the user to choose among Providers that support that institution. Provider support and Provider availability are separate concepts. Support means the Provider can connect that institution according to Provider-owned institution discovery, registry metadata, or adapter capability. Availability means the local `money` configuration has the required credentials for that Provider. Ray Finance determines configured Providers from local config/env fields such as Plaid client ID plus secret or Bridge client ID plus client secret; `money` should use the same local configured-state idea, adapted to its explicit `env:` config model. The Provider selection UI should show all Providers known to support the selected institution, including unavailable Providers marked as missing credentials, then block selection of unavailable Providers with generated configure guidance. Provider-specific commands such as `money providers plaid link` and `money providers bridge link` remain available for scripts, tests, and debugging.

Institution-first search is adapter-owned. Plaid uses `/institutions/search` with the configured products and country codes. A Provider that cannot support programmatic institution discovery in the current adapter is not invented into the institution-first result list; it remains reachable through its explicit provider command, such as `money providers bridge link`, with provider-specific prompts or flags. This keeps `money link` honest: every selectable provider/institution pairing must come from provider discovery, checked registry metadata, or an adapter capability that can actually link that institution.

Linking does not automatically run the first sync. A successful link stores the Provider Item and tells the user how to run `money sync` using command-registry/help-derived text. This preserves the project rule that networked data fetching happens only after explicit user direction.

Linking the same institution through more than one Provider creates separate Provider Items. The store should preserve provider/item/account provenance in IDs or source fields so account and transaction identity cannot collide across Providers. Future merge/dedupe commands must be dry-run first and require explicit user review before changing canonical data.

Transaction identity is source-deterministic in the first version. Provider transactions are idempotent by Provider Item plus provider-native transaction ID, such as `(provider_item_id, provider_transaction_id)`. Manual and imported transactions use local IDs plus explicit source metadata. Demo is a runtime environment, not a source kind; demo fixtures should still use provider, manual, or import source semantics. Do not automatically merge transactions through heuristics such as amount, date, merchant, or account similarity; those rules belong in a future review-based reconcile flow.

Read contracts show all unmerged Provider data. `accounts.list` and transaction contracts expose provenance through a uniform `source` object so callers do not infer source from scattered fields. The source object uses `kind` values `provider`, `manual`, or `import`. First-version stable contracts always include `provider`, `provider_item_id`, `provider_external_item_id`, `institution_id`, `provider_account_id`, `provider_transaction_id`, `import_source_id`, and `import_batch_id`; fields that do not apply to that source kind are `null`. `provider_item_id` is the local `pi_...` Provider Item ID. `provider_external_item_id` is the provider-native item identifier, such as Plaid `item_id` or a Bridge item ID. For Provider-sourced records, provider-native account and transaction object IDs are stable provenance fields. Provider tokens, credentials, full raw payloads, full account numbers, and routing numbers must never be exposed through `source`. Demo mode is identified at the envelope level with `meta.demo: true`.

## Configuration Boundary

Configuration belongs to `money`, not donor projects. Users may configure `money` by editing `~/.money/config.yaml`, loading `~/.money/.env`, setting environment variables, or running setup commands. Config sources complete each other through explicit value references instead of silently overriding one another. Config resolution must use `MONEY_*` names for `money` settings and provider-owned names for provider credentials; it must not silently read `~/.ray`, `RAY_*`, Monarch, donor config paths, or cwd `.env` files.

The concrete config loading contract is maintained in `docs/CONFIG.md`. Implementation and tests should treat that file as the source of truth for config path resolution, `env:` syntax, `.env` loading, database key decoding, and demo-mode config bypass behavior.

The default config path is `~/.money/config.yaml`; the default env companion is `~/.money/.env`. Alternate config paths require explicit selection through `--config` or `MONEY_CONFIG`.

When an alternate config path is selected, its default env companion is `.env` in the same directory as that config file. A config file may also explicitly set `env_file` to a specific secrets file. Relative `env_file` paths resolve relative to the config file's directory, never cwd. Cwd `.env` is never loaded implicitly.

Provider secret names should follow the provider's common official vocabulary. For Plaid, use `PLAID_CLIENT_ID`, `PLAID_SECRET`, `PLAID_ENV`, `PLAID_PRODUCTS`, `PLAID_COUNTRY_CODES`, and `PLAID_REDIRECT_URI`. For Bridge, use `BRIDGE_CLIENT_ID` and `BRIDGE_CLIENT_SECRET`. For MX, use `MX_CLIENT_ID` and `MX_API_KEY`. For Finicity, use `FINICITY_APP_KEY`, `FINICITY_PARTNER_ID`, and `FINICITY_PARTNER_SECRET`.

Plaid Dashboard login is a Plaid-specific credential bootstrap flow, not a generic Provider interface. `money plaid login` and `money providers plaid login` use the Plaid CLI-compatible Dashboard OAuth path to fetch API keys, then write only `PLAID_CLIENT_ID` and the selected environment's `PLAID_SECRET` through the normal Provider config writer. Dashboard OAuth tokens are stored separately as `plaid-dashboard-auth.json` beside the resolved config file with the same local-user file security expectations as `.env`. If Plaid rejects or changes the private Dashboard contract, the command fails explicitly and tells the user to configure Plaid manually; it must not try alternate private clients, scraping, or hidden fallback behavior.

Plaid defaults to the `transactions` product. Additional products such as `investments` and `liabilities` require explicit configuration and should not expand stable read contracts until their local data model is ready. Recurring transaction streams should be synced when the configured Provider returns them; otherwise `recurring.list` returns an empty local result with a normal success envelope.

Bridge link creates or reuses a Bridge external user ID without creating a `money` user account. By default `money` generates and stores the external user ID in Provider Item state during link. Advanced commands may accept an existing external user ID for reconnect or migration cases.

Provider link flows may use a short-lived localhost callback helper when required by the Provider, such as Plaid Link. This helper is not a required persistent server, local API, daemon, or background service. It should bind only for the active link session, use a random state value, shut down after completion or timeout, and fail explicitly when the callback cannot be started. Interactive link commands should follow GitHub CLI's browser flow: print the authorization URL and wait for the user to press Enter before opening the browser. If the user does not press Enter, nothing is opened and the printed URL remains usable for manual/headless handling. `--no-open` suppresses browser opening for SSH, cron, and headless environments while still printing the URL.

Plaid Link in CLI mode is implemented as a short-lived local Link page, not as a raw Plaid redirect URL. The command creates a Plaid `link_token`, starts a localhost callback/page server, prints the local URL, waits for Enter, then opens that local URL unless `--no-open` is set. The page loads Plaid Link from Plaid's CDN with the `link_token`; on `onSuccess`, it posts a `success` callback containing the `public_token`, Plaid institution metadata, selected account metadata, and random state back to the localhost helper. On `onExit`, it posts either a `cancel` callback or an `error` callback with Plaid's error type/code/message plus request and link-session metadata when available. The CLI validates state and same-host origin, exchanges only success callbacks for access tokens, stores only the encrypted access token and mapped metadata, then shuts down the helper. Cancel and Link error callbacks do not exchange a token. The helper must not expose general API endpoints or serve real financial data.

Plaid Link consent product options are explicit configuration/flag inputs. `products` remains the initial product set. `additional_consented_products`, `required_if_supported_products`, and `optional_products` may be configured or passed to Plaid link commands to shape consent without expanding local stable read contracts. Unsupported Plaid product names fail validation before calling Plaid.

Bridge link uses the same browser ergonomics but does not pretend to be Plaid Link. The Bridge adapter creates a connect session, prints the connect URL, waits for Enter before opening unless `--no-open` is set, then polls or checks provider item state according to Bridge's API. If Bridge institution discovery is unavailable in the first adapter, `money link` omits Bridge from institution-first choices and `money providers bridge link` remains the supported path.

Secret references use explicit `env:` objects so config loading has no magic environment lookup. Interactive setup and `money providers configure <provider>` must write secrets through `.env` plus YAML `env:` references, not direct YAML secret values. Secret entry prompts display mask characters such as `••••` instead of the raw value while the user types. Direct credential values in `config.yaml` are tolerated only when the user manually edits the file; config loading and doctor should emit a warning recommending `env:` for secrets.

Setup can complete without Provider credentials, leaving `money` usable for local reads, demo mode, and manual data but unable to link or sync real Provider data. Provider credential setup is optional and should default to skip. Setup Provider selection and review-workflow choices use the standard human-mode arrow-key selector, not numbered menus or raw yes/no text entry. Setup should not ask advanced path questions by default; advanced users can edit `config.yaml` or run with explicit `--config`/`MONEY_CONFIG`. Setup completion must show the saved config path, env path, and database path. When the user skips Provider credentials, setup must clearly state that sync/link commands will not work until credentials are added and must show how to find the current configure command using help-style output generated from the command registry, not hard-coded provider command strings.

Provider credentials are added through `money providers configure <provider>`. The command follows the same secret handling as setup: write secret values to the resolved env file, write or update config references to those env variable names, and never print secret values. Confirmation, summary, JSON, diagnostics, and errors may show env variable names and boolean secret status only; they must not show raw secret values or reversible partial previews. Existing env vars are not overwritten by default. Human mode asks before replacing an existing value, defaulting to No through the standard arrow-key selector. JSON and non-interactive modes require explicit `--force` to overwrite. If the user keeps an existing env var, the command may still update YAML `env:` references to point at that env var, then rely on doctor diagnostics to validate it. After writing, it should run the shared doctor checks and display only the configured Provider's diagnostics plus global blocking diagnostics such as config or store failures. It must not duplicate doctor rules or show unrelated Provider warnings. Help, setup hints, and doctor remediation text must derive Provider command examples from the registered Provider list so adding or removing Providers cannot leave stale setup text. `money doctor` reports whether the current configuration can read locally, link Providers, and sync.

Interactive Provider configuration must be atomic per Provider. If required fields for that Provider are missing or invalid during configuration, do not save a partial Provider credential block. The user can rerun `money providers configure <provider>` to complete it later.

Config writers should keep failure behavior simple and auditable. Use safe single-file writes where practical, but do not attempt complex cross-file rollback between `.env` and YAML. If one file write succeeds and a later write fails, stop immediately and report the exact file that was written, the file that failed, and the command/help path to rerun or repair manually. Do not print secret values in failure output.

`money doctor --json` is supported for agents and scripts, but it is not one of the first stable read contracts. It should report config path, database path, encrypted store readiness, local-read availability, Provider credential status, linked Provider Item counts, sync availability, and configuration warnings such as direct secrets in YAML.

Human `money doctor` output is grouped for scanning: `Config`, `Store`, `Providers`, `Links`, `Sync`, and `Warnings`. Individual checks use `ok`, `warn`, or `error` status labels. JSON doctor output uses structured objects and must not require callers to parse human headings.

Overly broad `.env` permissions are a warning, not an error, when they do not prevent use. The warning should state that secrets file permissions are broader than recommended and that `money doctor --fix` can repair them. On Unix-like platforms, `doctor --fix` tightens the secrets file to owner read/write only (`0600`). On platforms without POSIX chmod semantics, report a warning rather than emulating permissions. If permission repair fails, report it through the existing shared config write/fix failure handling without adding a separate error-code family unless implementation needs it.

First-version permission checks focus on the secrets env file, not `config.yaml`. Interactive commands should not write secrets directly into YAML. If a user manually places direct secrets in `config.yaml`, doctor reports the direct-secret warning; it does not need a separate YAML file-permission check in the first version.

Missing Provider credentials are a `warn`, not an `error`, because local reads, demo mode, and manual data remain usable. A Provider becomes `error` only when the user attempted to configure it and the configured credentials are malformed, incomplete, or fail validation. Setup must not duplicate these rules; after setup writes configuration, it runs the shared doctor checks and displays the returned diagnostics.

`money doctor` has no side effects by default: it does not create a database, run migrations, generate secrets, or rewrite config. `money doctor --fix` is the explicit repair mode and executes supported non-destructive repairs directly in human mode. It must be idempotent and must not damage a valid existing configuration. It only creates missing files or fills missing local basics; it must not overwrite user-set values, remove unknown fields, rewrite credentials, change file paths that are already set, or normalize stylistic formatting.

The first `doctor --fix` scope is limited to local configuration repair: create the `~/.money/` directory, create missing `config.yaml` and `.env` skeletons, generate a missing `MONEY_DB_ENCRYPTION_KEY` into `.env`, and fill a missing database path with the default. It must not migrate donor config, move or re-encrypt databases, modify Provider credentials, purge data, link accounts, sync, or make destructive edits. `--dry-run` may show the non-destructive repair plan without writing.

`money setup` and `money doctor --fix` may both create the same basic local configuration values, but their responsibilities are different. Setup is interactive guided configuration and may ask Provider and review-workflow questions. Doctor fix is non-interactive repair and only fills missing local configuration basics without changing existing credentials.

Generated database encryption keys must have at least 256 bits of cryptographic randomness. Setup and doctor fix should store the key as `MONEY_DB_ENCRYPTION_KEY`, preferably base64url without padding, and adapt it internally to the selected encrypted SQLite implementation. The key must not be derived from machine identifiers, must not use a weak default, and must not be printed in full in JSON output.

Generated database encryption keys are exactly 32 random bytes encoded as base64url without padding. Config loading decodes the value back to exactly 32 bytes before opening the store. Any malformed value or decoded length other than 32 bytes is a configuration error.

After setup writes configuration, human output should summarize paths and secret creation without revealing secret values, for example `Wrote ~/.money/config.yaml`, `Wrote ~/.money/.env`, and `Generated MONEY_DB_ENCRYPTION_KEY`. JSON setup output should include paths and booleans such as `secret_created: true`, never raw secret values.

After setup finishes writing configuration, it should create or open the encrypted SQLite database, run migrations, then run the same checks as `money doctor` internally against the real environment and report the returned diagnostics. This gives read commands a stable encrypted empty-state immediately after setup. If database open or migrations fail, keep the written config/env files, stop immediately, report the DB or migration failure and the files already written, and let the user repair and rerun setup or doctor. If an existing DB file cannot be opened, setup must not create a replacement DB at that path. Do not roll back config/env for DB initialization failures. Setup should not suggest demo mode as a next step; demo mode remains discoverable through help as an optional sandbox, not part of normal setup completion.

```yaml
database:
  path: ~/.money/data/money.db
  encryption_key:
    env: MONEY_DB_ENCRYPTION_KEY

providers:
  plaid:
    client_id:
      env: PLAID_CLIENT_ID
    secret:
      env: PLAID_SECRET
    environment: sandbox
    products: [transactions]
    country_codes: [US]
  bridge:
    client_id:
      env: BRIDGE_CLIENT_ID
    client_secret:
      env: BRIDGE_CLIENT_SECRET
```

## Implementation Tooling

Use Cobra for CLI command routing, flags, aliases, help generation, and shell completion. Keep Cobra in the CLI package only; core, store, contracts, and providers must not import Cobra.

Use Go's standard `testing` package for the default test suite. Add `testify` only when repeated assertion or require boilerplate starts obscuring contract tests; do not introduce it preemptively.

Use `github.com/olekukonko/tablewriter` for plain human-mode tables. Use ANSI color only as an auxiliary signal, and keep JSON mode independent of terminal rendering.

Use `charm.land/huh/v2` for human-mode arrow-key selectors and forms. Choice prompts must still be wrapped behind a small internal prompt interface so JSON and non-TTY paths can return validation errors without importing terminal UI code into core logic.

The initial external dependencies are therefore:

- `github.com/spf13/cobra`
- `github.com/ncruces/go-sqlite3`, `github.com/ncruces/go-sqlite3/driver`, and `github.com/ncruces/go-sqlite3/vfs/adiantum`
- `github.com/olekukonko/tablewriter`
- `charm.land/huh/v2`

Do not add Viper for config loading; `money` has explicit config resolution rules and should not inherit implicit environment override behavior.

## Store Schema

The first migration contract is defined in `docs/SCHEMA.md`. Implementation work that changes table names, columns, source mapping, money representation, or migration behavior must update that document, migration files, command contracts, and tests together.

## First Stable Contracts

### accounts.list

Command:

```bash
money accounts list --json
```

Contract goal: return visible financial accounts with canonical IDs, account type, subtype, balances, currency, source, and update time.

The contract must include Provider provenance so duplicate-looking accounts from separate Provider Items remain distinguishable.

Human `accounts list` defaults to scan-friendly columns: `NAME`, `TYPE`, `BALANCE`, `AVAILABLE`, `CURRENCY`, `SOURCE`, and `UPDATED`. Provider raw IDs and full provenance are omitted from the default human table and can be shown with `--verbose`. JSON output always includes the full stable account contract, including the `source` object.

`current_balance` is signed by financial-position contribution: assets are positive, liabilities are negative. Credit capacity is separate from account value. `current_balance` must not include credit limits or available credit; those values need separate fields if exposed.

`available_balance` is only for cash-like available money. Credit borrowing capacity must be exposed with credit-specific fields such as `available_credit` or omitted.

### transactions.list

Command:

```bash
money transactions list --json
```

`money tx list --json` is a CLI alias for the same contract.

Contract goal: return deterministic recent or filtered transaction results by date range, account, category, merchant, tag, review status, pending status, recurring status, and pagination.

Human `transactions list` and `transactions search` default to scan-friendly columns: `DATE`, `ACCOUNT`, `MERCHANT`, `AMOUNT`, `CATEGORY`, and `STATUS`. `STATUS` is a compact flag list containing values such as `pending`, `review`, and `removed`; it is empty when no flags apply. `--verbose` may show local IDs, source provenance, note, tags, provider category, and other diagnostic fields. JSON output always includes the full stable transaction contract.

Human money output must be identifiable without color. Transaction amounts include explicit signs: positive inflows render with `+`, negative outflows render with `-`, and zero renders without a misleading sign. When color is available, follow US stock-market convention as an auxiliary cue: green for positive/inflow/up values and red for negative/outflow/down values. Color must never be the only signal.

### transactions.search

Command:

```bash
money transactions search <query> --json
```

`money tx search <query> --json` is a CLI alias for the same contract.

Contract goal: return deterministic transaction search results by text query, date range, account, category, merchant, tag, review status, pending status, recurring status, and pagination. The query is required; filter-only transaction retrieval belongs to `transactions.list`.

Transaction `amount` uses Monarch-style signed semantics: positive values are inflows and negative values are outflows. Provider adapters must normalize provider-native signs before records reach stable contracts.

Pending transactions are included by default and must expose `pending: true` in the contract. The command should support explicit pending filters rather than hiding pending records.

Default transaction ordering is `date DESC, pending DESC, id ASC` until a richer posted timestamp exists.

Split transactions are not part of the first `money` transaction read contract. Ray Finance does not model split transaction state in its transaction schema or query projection. Monarch supports split read/write through separate remote commands, but `money` should add split support later only after a local split schema, dry-run/confirm write flow, and amount validation are designed.

### categories.list

Command:

```bash
money categories list --json
```

Contract goal: return deterministic transaction categories and their groups.

### tags.list

Command:

```bash
money tags list --json
```

Contract goal: return deterministic transaction tags.

Tags are local annotations. Providers do not supply tags; Provider categories remain separate from local tags. Monarch imports may populate local tags because they are user-owned migrated data.

Transaction read contracts expose tags as both `tag_ids` and `tags`. `tag_ids` is an array of local `tag_...` IDs for filtering and scripting. `tags` is an array of readable objects with at least `id` and `name` for agent ergonomics. Both fields describe the same local annotations; Provider data does not create or own tags.

Transaction notes are a single local `note` string field in the first version, with `null` when absent. Provider sync and imports must not overwrite an existing local note unless a later explicit import/reconcile operation is designed for that purpose. Rich text, comment threads, and attachments are out of scope for the first transaction contract.

Provider sync must preserve local annotations. Provider-owned fields may be updated from sync results, but local tags, notes, review state, user overrides, and custom fields must not be overwritten by a provider refresh. If a field can be both provider-supplied and user-edited, the store needs provenance or override state before sync can update it safely.

Category uses a dual-field model. Provider sync writes `provider_category` and `provider_subcategory`. User recategorization writes local category fields. Transaction read contracts expose top-level `category_id` as the local `cat_...` ID or `null`, `category_name` as the effective display name, and `category_source` as `local`, `provider`, `import`, or `none`. Provider raw category fields remain separate so agents can inspect provider classification without treating it as the local category.

Provider sync must not create local categories from provider category names. When only provider classification exists, `category_source` may be `provider`, `category_name` may reflect the provider display category, and `category_id` remains `null`. Local categories are created only by the user, imports, setup/demo fixtures, or a future explicit category import/sync command.

Review state is a local optional workflow. First-version transaction contracts expose a single boolean `needs_review`, defaulting to `false`. Setup asks one default-off selector question: `Mark newly synced transactions as needing review?`, with No selected by default. When enabled, only newly synced transactions receive `needs_review: true`; existing transactions are unchanged. Provider sync must not treat review state as provider-owned. Review status enums, assignees, workflow history, and review comments are out of scope for the first contract.

### recurring.list

Command:

```bash
money recurring list --json
```

Contract goal: return deterministic recurring transaction streams or upcoming recurring items available from local data.

## Non-Negotiables

- No hidden compatibility fallbacks.
- No embedded AI runtime.
- No hosted OpenRay/Money service dependency.
- No read command should require provider credentials.
- No command should open real financial data from plaintext SQLite.
- No provider command should require a registered `money` account, subscription, managed proxy key, or AI key.
- No command should silently read donor config paths or donor environment variable names.
- No provider-specific fields should leak into stable contracts unless intentionally namespaced.
- Donor repositories remain references, not mixed source.
