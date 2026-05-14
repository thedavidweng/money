# Plaid Dashboard Login Implementation Plan

Date: 2026-05-13

## Goal

Make Plaid setup in `money` feel close to the official Plaid CLI while preserving `money`'s local-first boundaries.

The target user experience:

- During `money setup`, after choosing Plaid, the user can choose either:
  - Open a browser, sign in to the Plaid Dashboard, and let `money` fetch Plaid API keys automatically.
  - Manually paste `client_id` and `secret`, preserving the existing BYOK path.
- The user can also run `money plaid login` directly, without going through setup, to bootstrap Plaid credentials through the browser.
- The canonical provider command remains available as `money providers plaid login`; `money plaid login` is a convenience alias for the Plaid-specific workflow.

## Constraints

- Do not copy Plaid CLI source or donor code.
- Do not add AI chat, hosted billing, telemetry, a persistent web server, or a managed provider proxy.
- Do not silently fall back from Dashboard login to manual credentials. If Dashboard login fails, return an explicit error and tell the user to run the manual command if they choose.
- Keep Dashboard OAuth and Dashboard API behavior Plaid-specific. Do not add it to the generic Provider interface.
- Keep provider credentials in `money`'s config model: secrets in `.env`, YAML using explicit `env:` references.
- Do not print Plaid API secrets, Dashboard OAuth tokens, or reversible partial previews.
- Treat Plaid Dashboard endpoints as private Plaid CLI-compatible endpoints. The implementation must fail clearly if Plaid rejects non-official clients or changes the contract.
- Plaid Dashboard login is best-effort. It reuses the Plaid CLI-compatible OAuth client behavior observed during donor research, and it may stop working if Plaid changes Dashboard backend behavior.
- If Plaid locks down or rejects the `plaid-cli` OAuth client path, stop the Dashboard login flow immediately. Do not try alternate private clients, hidden fallbacks, or scraping. Return a typed error that points users to manual `money providers configure plaid`.

## Current State

Already present:

- `money setup` creates config, `.env`, encryption key, and encrypted SQLite database.
- `money providers configure plaid` writes `PLAID_CLIENT_ID` and `PLAID_SECRET` into the resolved `.env` and YAML `env:` references into config.
- `money link <institution-query>` and `money providers plaid link` create a Plaid Link token, start a short-lived localhost helper, open Plaid Link, receive the public token, exchange it, and store the linked Provider Item in the encrypted store.

Missing:

- Plaid Dashboard OAuth.
- Plaid Dashboard team selection.
- Automatic Plaid API key fetch.
- `money plaid login` top-level convenience command.
- Setup branch that lets the user choose automatic Plaid Dashboard login instead of manual key entry.

## Command Surface

### New Commands

```text
money plaid login
money plaid logout
money providers plaid login
money providers plaid logout
```

`money plaid login` and `money providers plaid login` call the same implementation.

This is an intentional asymmetric top-level namespace. Plaid gets `money plaid ...` because the Dashboard OAuth bootstrap is Plaid-specific and modeled from the Plaid CLI donor; this does not imply future Providers must receive matching top-level namespaces unless they have similarly useful Provider-specific workflows.

Initial login flags:

```text
--no-open          Print the Dashboard OAuth URL without opening a browser.
--team <selector>  Select a team by ID, client ID, name, or 1-based index when available.
--environment      Plaid environment to write into money config. Default: sandbox.
--force            Overwrite existing PLAID_CLIENT_ID / PLAID_SECRET values.
```

`--json` implies automation mode and behaves as `--no-open`: print the authorization URL to stderr, do not wait for Enter, do not open the browser, wait for the localhost callback until timeout, and write exactly one final JSON envelope to stdout. Do not implement a JSON mode that prompts, blocks on stdin, or mixes progress text into stdout.

Later optional Dashboard commands, only if they are still useful after `login` is working:

```text
money plaid teams list
money plaid teams use <selector>
money plaid keys fetch
```

Do not implement `teams` and `keys fetch` until the core `login` flow proves useful. They expose more of Plaid's private Dashboard API surface and are not required for the setup experience.

Related follow-up plans:

- `docs/plans/2026-05-13-plaid-link-hardening.md`
- `docs/plans/2026-05-13-plaid-sandbox-link.md`

Those are deliberately outside this first implementation slice. This plan should land Dashboard login, setup UX, prompt infrastructure, contracts, and docs without also changing Plaid Link product behavior or adding Sandbox direct linking.

## Setup UX

When `money setup` detects Plaid is unconfigured and the user chooses Plaid, show a second Plaid-specific choice using the project's standard Hermes-style selector implemented with `charm.land/huh/v2`. Do not use a numbered menu.

```text
How do you want to configure Plaid?

> Sign in with Plaid Dashboard and fetch API keys automatically
  Paste client ID and secret manually
  Skip Plaid for now
```

Behavior:

- Dashboard login choice calls the same flow as `money plaid login`.
- Manual credential choice calls the existing provider configuration flow.
- Skip choice returns to setup without writing a partial Plaid provider block.
- Non-interactive usage must provide flags or return a validation error. It must not choose a path implicitly.
- The existing `runSetupWizard` provider-selection prompt currently uses a numbered menu. Migrate that prompt to the same `huh/v2` selector while adding the Plaid method selector, so this plan does not preserve a known anti-pattern.

The setup flow should finish with:

- Config path.
- Env path.
- Database path.
- Whether Plaid credentials are present.
- Next command: `money link <institution-query>` or `money providers plaid link`.

## Dashboard OAuth Flow

Create a Plaid-specific Dashboard login module, not a generic provider abstraction.

Suggested package boundary:

```text
internal/plaidlogin/
  oauth.go          PKCE, auth URL construction, token exchange inputs.
  callback.go       Short-lived localhost OAuth callback server.
  dashboard.go      Dashboard API HTTP client.
  credentials.go    Convert fetched Dashboard keys into money config writes.
```

OAuth steps:

1. Require an existing base `money` config. If `money plaid login` runs before `money setup`, do not initialize config implicitly.
   - Human mode: print guidance to run `money setup`, then exit with `BASE_CONFIG_MISSING`.
   - JSON mode: return a normal error envelope with `BASE_CONFIG_MISSING`.
   - Detection must resolve the intended config path, stat/read that file, and classify `errors.Is(err, fs.ErrNotExist)` as `BASE_CONFIG_MISSING`. Do not call `config.Setup()` from `money plaid login`; `config.Setup()` creates the config skeleton and would hide the missing-base-config error.
   - Do not rely only on full `config.Load()` for this first check. `config.Load()` resolves every `env:` reference and can fail when Plaid credentials are exactly what `money plaid login` is trying to repair. Add or reuse a config helper that resolves the config path and `env_file` path from raw YAML without requiring Provider secrets to resolve.
2. Start a short-lived localhost callback server with separate bind and redirect hosts:
   - `bindHost = "127.0.0.1"` for the listener, so the server binds only loopback without depending on localhost DNS or IPv6 behavior.
   - `redirectHost = "localhost"` for the OAuth redirect URI, matching the Plaid CLI donor evidence and avoiding strict `redirect_uri` mismatch risk.
3. Generate a random state and PKCE verifier.
4. Build the Dashboard OAuth URL:
   - Auth URL: `https://dashboard.plaid.com/oauth/authorize`
   - Token URL: `https://api.dashboard.plaid.com/oauth/token`
   - Client ID: Plaid CLI-compatible Dashboard OAuth client, currently observed as `plaid-cli`
   - Redirect URL: `http://localhost:<port>/oauth/callback`
5. Print the URL. If `--no-open` is set or `--json` is set, never open a browser.
6. Otherwise follow the same GitHub-style browser flow used by `money link`: wait for the user to press Enter, then open the browser.
7. Wait for `/oauth/callback?code=...&state=...`.
8. Reject missing code, OAuth error, wrong state, duplicate callback, wrong method, and timeout.
9. Exchange the authorization code using PKCE.
10. Store Dashboard auth only after token exchange succeeds. At this point `team_id` and `client_id` may still be absent.
11. After team selection and key fetch, update the same Dashboard auth file with selected `team_id` and Plaid `client_id` when available.

## Plaid CLI Compatibility Boundary

This feature is based on Plaid CLI-compatible Dashboard behavior observed from the installed Plaid CLI version `20260507-4d1b0ca0`.

Known donor evidence:

- OAuth authorize URL: `https://dashboard.plaid.com/oauth/authorize`.
- OAuth token URL: `https://api.dashboard.plaid.com/oauth/token`.
- OAuth client ID: `plaid-cli`.
- Redirect URI string: `http://localhost:<port>/oauth/callback`.
- Token exchange body uses `grant_type=authorization_code`, `code`, `redirect_uri`, `client_id`, and PKCE `code_verifier`.
- Refresh body uses `grant_type=refresh_token` and `refresh_token`.
- Dashboard teams endpoint path: `/cli/teams/list`.
- Dashboard keys endpoint path: `/cli/keys/fetch`.
- Keys payload is expected to include `client_id` plus environment secrets for at least `sandbox` and optionally `production`.

Details still requiring live or binary re-verification in Phase 1:

- Whether `/cli/keys/fetch` is currently `GET` or `POST`.
- Exact team response field names.
- Exact keys response field names and nesting.
- Exact 401 body shape for `team_selection_required`, `api_keys_fetch_required`, and any new Dashboard auth errors.

Implementation requirements:

- Add a Plaid CLI compatibility version constant in `internal/plaidlogin/`, currently `20260507-4d1b0ca0`. The exact Go identifier may be exported or unexported depending on whether `doctor` or `login --json` needs to report it.
- Add a short source comment beside that constant explaining that the Dashboard API contract is private and was last verified against the Plaid CLI version above.
- Document in `README.md` and `docs/GETTING_STARTED.md` that Dashboard login is best-effort and manual provider configuration remains the durable fallback.
- If `/cli/teams/list`, `/cli/keys/fetch`, or OAuth token responses cannot be parsed into the expected typed structures, return `DASHBOARD_CONTRACT_CHANGED`.
- If Plaid rejects the OAuth client or returns an authorization failure that indicates the CLI-compatible path is blocked, return `PLAID_DASHBOARD_LOGIN_REJECTED`.
- Error messages for both cases must explicitly tell the user to run `money providers configure plaid` manually and include the Plaid keys URL from `config.PlaidSpec.HelpURL`.
- Do not retry with different undocumented endpoints, alternate client IDs, or browser scraping.

## Dashboard Token Storage

Persist Dashboard OAuth state separately from provider API credentials:

```text
<dirname(resolved config path)>/plaid-dashboard-auth.json
```

File mode: `0600`.

Path rules:

- Default profile: `~/.money/plaid-dashboard-auth.json`.
- Named profile: `~/.money/profiles/<profile>/plaid-dashboard-auth.json`.
- Explicit `--config /tmp/foo/config.yaml`: `/tmp/foo/plaid-dashboard-auth.json`.
- Resolution follows the same `--config`, `MONEY_CONFIG`, and `--profile` path rules as normal config loading.
- `money doctor` warns when the auth file exists with permissions broader than `0600`, matching `.env` permission behavior.
- Like `.env`, POSIX permission warnings/fixes are skipped on Windows.

Schema:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "expires_at": "2026-05-13T15:04:05Z",
  "team_id": "...",
  "client_id": "..."
}
```

`team_id` and `client_id` are optional until team selection and key fetch complete. A failed key fetch may leave a valid Dashboard auth file without Provider API credentials; that is acceptable because Dashboard auth is bootstrap state, not the Provider credential contract.

Rationale:

- Dashboard auth is bootstrap state. It must be readable before opening the encrypted SQLite store, otherwise `money plaid login` would need to unlock or initialize the finance database before it can fetch API credentials.
- Dashboard tokens have TTL and refresh-rotation semantics, which do not match the static `.env` plus YAML `env:` Provider credential model.
- The auth file uses the same local-user file-security level as `.env`: resolved beside config, written `0600`, never printed, and warned on broad permissions.
- The Provider adapter only needs `PLAID_CLIENT_ID`, `PLAID_SECRET`, `PLAID_ENV`, `PLAID_PRODUCTS`, `PLAID_COUNTRY_CODES`, and `PLAID_REDIRECT_URI`.
- Keeping Dashboard auth outside YAML avoids adding token fields to the stable Provider config contract.
- `money plaid logout` can delete this file without touching API keys or linked Provider Items.

## Team and Key Fetch Flow

After OAuth token exchange:

1. Call Dashboard teams list.
2. If no teams are returned, fail explicitly and write no Plaid credential changes.
3. If one team is returned, select it.
4. If multiple teams are returned:
   - Use `--team` if provided.
   - Otherwise prompt interactively with the project's standard `huh/v2` selector.
   - In non-interactive mode, return a validation error listing team selectors.
5. Fetch API keys for the selected team.
6. Choose the Plaid environment to configure:
   - Default to `sandbox`.
   - Honor `--environment production`.
   - Reject any other value with validation error `INVALID_ENUM`.
   - If the selected environment has no secret, return `PLAID_ENVIRONMENT_NOT_PROVISIONED`.
7. Write `PLAID_CLIENT_ID` and `PLAID_SECRET` through the shared Provider credential writer described below.
8. Update Plaid config fields:
   - `environment`: from `--environment`, defaulting to `sandbox`.
   - `products`: from flag if provided, otherwise `config.PlaidSpec.OptionalFields`.
   - `country_codes`: from flag if provided, otherwise `config.PlaidSpec.OptionalFields`.
   - `redirect_uri`: from flag if provided, otherwise `config.PlaidSpec.OptionalFields`.
9. Run shared diagnostics and show only global blockers plus Plaid status.

Do not write partial provider credentials. If team selection or key fetch fails, leave existing Plaid credentials unchanged unless `--force` has already been validated and a complete fetched key set is ready.

Schema drift behavior is governed by the Plaid CLI compatibility boundary above. Missing required response fields, changed token shape, unrecognized team shape, or missing selected environment secrets are not recoverable by guessing.

Dashboard does not provide `products`, `country_codes`, or `redirect_uri`; `money` writes those from flags or from local Plaid defaults. Do not infer these values from Dashboard responses.

Team selector rules:

- Preserve the Dashboard `/cli/teams/list` response order for display and 1-based index matching.
- Team ID and client ID selectors use exact matching.
- Team name selectors use case-insensitive full-string matching.
- If one selector matches multiple teams or multiple selector modes, return an ambiguity validation error instead of choosing the first match.

## Config and Secret Rules

- Reuse `config.ConfigureProvider` for final credential writes only after its current overwrite and env-path behavior is made compatible with this flow.
- Current implementation trap: `config.ConfigureProvider` currently skips existing env vars when `force=false` and writes the default `.env` beside config. Before Dashboard login depends on it, either update that function or add a small shared helper so provider credential writes:
  - resolve config path and env path without requiring existing Provider secrets to load,
  - honor `cfg.EnvPath`, including explicit `env_file`,
  - check existing `PLAID_CLIENT_ID` / `PLAID_SECRET` before writing,
  - ask or fail before overwrite according to the rules below,
  - report `keys_written` accurately, and
  - never claim success when fetched keys were skipped because old env values already existed.
- The helper must still write YAML `env:` references and never write direct YAML secrets.
- Existing `PLAID_CLIENT_ID` / `PLAID_SECRET` are not overwritten unless `--force` is set or the user explicitly confirms in human mode.
- JSON and non-interactive mode require `--force` to overwrite.
- If existing Plaid API credentials are present and the selected environment matches the current config environment, `money plaid login` may refresh Dashboard auth and preserve API keys without `--force`; report this as `credential_action: "preserved_existing"` and `keys_written: 0`.
- If existing Plaid API credentials are present and the selected environment differs from current config, or the command would otherwise replace `PLAID_CLIENT_ID` / `PLAID_SECRET`, require `--force` in JSON/non-interactive mode or explicit human confirmation before OAuth starts.
- If loaded config has effective `read_only: true` or `MONEY_READ_ONLY=1`, stop before OAuth or file writes with a read-only safety error; do not fetch tokens that cannot be persisted.
- Read-only detection may require a partial raw-config load when full `config.Load()` fails due to missing Provider credentials; `MONEY_READ_ONLY=1` is sufficient to block immediately, and raw `read_only: true` must also block before OAuth.
- `money plaid login --json` must keep stdout as one final JSON envelope. Progress messages and browser URL go to stderr. The CLI wiring must pass or use a stderr writer for this command, for example via Cobra's `cmd.ErrOrStderr()`, because the current providers command constructor only receives stdout.

Repeated login semantics:

- Re-running `money plaid login` always starts a new browser OAuth flow and overwrites the Dashboard auth file after successful token exchange.
- Token refresh happens inside the Dashboard client when a Dashboard API call needs it; it is not the user-visible behavior of the `login` command.
- Re-running `money plaid login` does not overwrite existing `PLAID_CLIENT_ID` or `PLAID_SECRET` unless the fetched key write is explicitly allowed.
- Human mode asks before replacing existing Plaid API credentials, defaulting to No through the standard `huh/v2` selector.
- JSON and non-interactive mode require `--force` before overwriting existing Plaid API credentials.
- `PLAID_SECRET` stores only the currently selected Plaid environment's secret. Switching environment requires `money plaid login --environment <sandbox|production> --force`; this overwrites the previous `PLAID_SECRET`. Keeping both Sandbox and Production secrets locally is out of scope until the Provider config contract explicitly supports per-environment secrets.

Logout semantics:

- `money plaid logout` deletes only `plaid-dashboard-auth.json`.
- If the auth file is already absent, logout succeeds with `dashboard_auth_removed: false`; this is idempotent and still preserves API keys.
- `money plaid logout` does not delete `PLAID_CLIENT_ID`, `PLAID_SECRET`, linked Provider Items, accounts, transactions, or sync state.
- Human output must say: `API keys remain in <env_path>. To remove them, edit the file or run money providers configure plaid with new values.`

## JSON Contracts and Errors

All new `--json` commands must use the standard envelope from `docs/CONTRACTS.md`: one stdout object with `ok`, `data`, `meta`, `warnings`, and `errors`.

`money plaid login --json` success data:

```json
{
  "provider": "plaid",
  "team_id": "team_...",
  "environment": "sandbox",
  "keys_written": 2,
  "credential_action": "written",
  "dashboard_auth_path": "/Users/you/.money/plaid-dashboard-auth.json",
  "next_command": "money link <institution-query>",
  "config_path": "/Users/you/.money/config.yaml",
  "env_path": "/Users/you/.money/.env"
}
```

Do not include secrets, tokens, masked secrets, refresh tokens, or reversible previews.

`money plaid logout --json` success data:

```json
{
  "provider": "plaid",
  "dashboard_auth_removed": true,
  "dashboard_auth_path": "/Users/you/.money/plaid-dashboard-auth.json",
  "api_keys_preserved": true,
  "env_path": "/Users/you/.money/.env"
}
```

Add these shared error codes to `docs/CONTRACTS.md` during implementation:

| Code | Category | Retryable | Exit code | Meaning |
| --- | --- | --- | --- | --- |
| `BASE_CONFIG_MISSING` | `config` | false | 3 | `money plaid login` was run before base config exists. |
| `NOT_LOGGED_IN` | `auth` | false | 3 | Dashboard-only operation needs Dashboard auth. |
| `TEAM_SELECTION_REQUIRED` | `validation` | false | 2 | Multiple teams exist and no interactive or `--team` selection was available. |
| `API_KEYS_FETCH_REQUIRED` | `auth` | true | 3 | Dashboard auth exists but API keys still need to be fetched. |
| `DASHBOARD_TOKEN_REFRESH_FAILED` | `auth` | false | 3 | Stored Dashboard refresh token cannot refresh; rerun `money plaid login`. |
| `DASHBOARD_CONTRACT_CHANGED` | `api` | false | 6 | Plaid Dashboard private response shape no longer matches the known contract. |
| `PLAID_DASHBOARD_LOGIN_REJECTED` | `auth` | false | 3 | Plaid rejected the CLI-compatible Dashboard OAuth path. |
| `PLAID_ENVIRONMENT_NOT_PROVISIONED` | `validation` | false | 2 | Dashboard returned no secret for the selected Plaid environment. |
| `READ_ONLY_VIOLATION` | `safety` | false | 4 | Dashboard login/logout would mutate local config, env, or auth files while read-only mode is enabled. |

In this repo's current code, "shared registry" means a central `docs/CONTRACTS.md` error-code section plus Go `const` names reused where errors are emitted. Do not invent a runtime map unless the implementation first introduces a small, obvious contracts-level constant block and updates existing call sites deliberately.

## Prompt Boundary

Add the prompt abstraction before changing setup UX:

- Add a small `internal/prompt` boundary around human-mode selectors.
- Use `charm.land/huh/v2` behind that boundary for arrow-key selectors.
- Keep JSON and non-TTY behavior prompt-free: required missing choices return validation errors.
- Provide fake prompt implementations for CLI/setup tests so tests do not depend on terminal UI or `bufio.Reader` menu parsing.
- Replace the existing `runSetupWizard` numbered provider menu through this boundary before adding the Plaid method selector.

## Out of Scope

- Plaid CLI analytics.
- Plaid CLI feedback submission through Dashboard API.
- Dashboard webhooks.
- Trial plan onboarding.
- Item limit display.
- Full per-environment Plaid credential vault in `money`.
- A persistent local server.
- Generic OAuth support for all Providers.

## Implementation Checklist

### Phase 1: Evidence and Contract Lock

- [x] Re-read `AGENTS.md`, `docs/ARCHITECTURE.md`, `docs/CONFIG.md`, `docs/CONTRACTS.md`, and this plan.
- [x] Re-check current Plaid CLI docs and the installed Plaid CLI help before coding, because Dashboard endpoints are drift-prone.
- [x] Confirm exact Dashboard OAuth endpoint strings and CLI-compatible client ID from current Plaid CLI behavior.
- [x] Add a Plaid CLI compatibility version constant with value `20260507-4d1b0ca0` and document the private compatibility boundary.
- [x] Confirm the manual fallback path and error messages include the Plaid keys URL.
- [x] Confirm listener bind host and OAuth redirect host remain separate: `127.0.0.1` for bind, `localhost` for redirect URI.

### Phase 2: Plaid Dashboard Login Module

- [x] Add Plaid Dashboard OAuth URL builder with PKCE and random state.
- [x] Add short-lived OAuth callback server.
- [x] Add token exchange client.
- [x] Add Dashboard token refresh using the persisted auth file.
- [x] Add Dashboard teams list client.
- [x] Add Dashboard keys fetch client.
- [x] Add typed errors for not logged in, team selection required, key fetch required, token refresh failed, Dashboard API contract changes, and Plaid Dashboard login rejection.
- [x] Add contract-drift tests for missing fields, unexpected fields, and unknown 401 Dashboard codes.
- [x] Add team selector logic with exact ID/client-ID matching, case-insensitive full-name matching, 1-based index matching, and ambiguity detection.
- [x] Add environment validation for `sandbox` and `production`, with `PLAID_ENVIRONMENT_NOT_PROVISIONED` when the selected environment has no fetched secret.

### Phase 3: Config Integration

- [x] Add atomic Dashboard auth file read/write/delete helpers.
- [x] Store Dashboard auth at `<dirname(resolved config path)>/plaid-dashboard-auth.json`.
- [x] Add doctor warning for Dashboard auth file permissions broader than `0600`, skipping POSIX permission checks on Windows.
- [x] Add or expose a config path/env-file resolver that can read raw config path metadata without resolving Provider secret env refs.
- [x] Add fetched-key-to-provider-config adapter.
- [x] Ensure Plaid API credentials still end up in the resolved `cfg.EnvPath` and YAML `env:` references.
- [x] Fix or wrap `config.ConfigureProvider` so Dashboard login honors explicit `env_file` and cannot silently skip fetched keys when existing env values are present.
- [x] Write `products`, `country_codes`, and `redirect_uri` from flags or `config.PlaidSpec.OptionalFields`; do not expect them from Dashboard.
- [x] Ensure direct YAML secrets are not written.
- [x] Ensure existing values require `--force` or explicit human confirmation.
- [x] Enforce read-only mode before OAuth, token persistence, credential writes, and logout deletion.
- [x] Add the new shared error codes to `docs/CONTRACTS.md` with category, retryable, and exit-code semantics.
- [x] Add Go `const` names for the new error codes where they are emitted; do not introduce a runtime registry unless it is a deliberate separate refactor.

### Phase 4: Prompt Boundary

- [x] Add `internal/prompt` with a small selector interface.
- [x] Add `charm.land/huh/v2` behind that interface for human mode.
- [x] Return validation errors rather than prompting in JSON or non-TTY mode.
- [x] Add fake prompt implementation for setup/login tests.
- [x] Migrate the existing `runSetupWizard` provider-selection prompt from numbered input to the prompt boundary.

### Phase 5: CLI Commands

- [x] Add `money providers plaid login`.
- [x] Add `money plaid login` alias command.
- [x] Add logout commands.
- [x] Wire `--no-open`, `--team`, `--environment`, `--force`, and `--json`.
- [x] Implement JSON live-login as `--no-open`: print the authorization URL to stderr, do not wait for Enter, do not open the browser, wait only for callback/timeout, and keep stdout as the final JSON envelope.
- [x] Ensure `money plaid login` fails with `BASE_CONFIG_MISSING` if setup has not been run.
- [x] Detect missing base config by resolving/statting the intended config file and checking `errors.Is(err, fs.ErrNotExist)`; do not call `config.Setup()` from `money plaid login`.
- [x] Ensure `money plaid login` can repair missing Plaid env refs instead of failing early because full `config.Load()` cannot resolve existing `providers.plaid.*` references.
- [x] Match `money link` browser ergonomics: print URL, wait for Enter, then open unless `--no-open`.
- [x] Ensure `money plaid login` always starts a new browser OAuth flow and only token refreshes inside Dashboard client calls.
- [x] Ensure environment switching requires `--force` when it overwrites existing `PLAID_SECRET`.
- [x] Implement repeated login and logout semantics exactly as documented above.
- [x] Keep errors explicit and categorized.

### Phase 6: Setup Integration

- [x] Add Plaid-specific method choice after selecting Plaid in setup.
- [x] Implement Plaid method choice through `internal/prompt`, not a numbered menu.
- [x] Route automatic choice to the login implementation.
- [x] Route manual choice to the existing provider configure implementation.
- [x] Ensure skip writes nothing.
- [x] Preserve the existing `!state.json && state.stdin != nil` gate around the setup wizard. In `--json` or non-TTY mode, `money setup` must skip the Plaid method selector entirely and behave like today: write base config and return success without prompting.
- [x] Ensure setup summaries and diagnostics reflect the chosen path.

### Phase 7: Tests

- [x] Unit-test OAuth URL construction, state, PKCE verifier use, and redirect URL.
- [x] Unit-test bind host and redirect URI host are intentionally different.
- [x] Unit-test callback server success, OAuth error, missing code, wrong state, duplicate callback, and timeout.
- [x] Unit-test Dashboard client request paths, auth headers, response parsing, and error classification with `httptest`.
- [x] Unit-test team selection by index, ID, client ID, and name.
- [x] Unit-test multi-team interactive selection through the shared `huh/v2` prompt boundary.
- [x] Unit-test ambiguous team selectors return validation errors.
- [x] Unit-test key fetch writes the selected environment only.
- [x] Unit-test selected environment without a secret returns `PLAID_ENVIRONMENT_NOT_PROVISIONED`.
- [x] Unit-test `products`, `country_codes`, and `redirect_uri` come from flags/defaults, not Dashboard.
- [x] Unit-test credential writes honor explicit `env_file`.
- [x] Unit-test login repairs a config whose Plaid `env:` refs exist but whose env values are missing.
- [x] Unit-test existing Plaid env vars require `--force` or human confirmation and cannot produce a misleading success with `keys_written: 0`.
- [x] Unit-test existing Plaid env vars with same environment can preserve API keys and report `credential_action: "preserved_existing"` plus `keys_written: 0`.
- [x] Unit-test read-only mode blocks login/logout before OAuth or file writes.
- [x] Unit-test fake Dashboard drift responses with missing fields, unexpected fields, and new 401 codes return typed errors instead of hidden fallback behavior.
- [x] Unit-test no partial writes on failed OAuth, failed teams, failed keys, or missing selected environment secret.
- [x] Unit-test `money plaid logout` deletes only `plaid-dashboard-auth.json`, is idempotent when the auth file is already absent, and preserves `PLAID_CLIENT_ID` / `PLAID_SECRET`.
- [x] CLI-test `money plaid login` and `money providers plaid login` against fakes.
- [x] Setup-test automatic Plaid login choice, manual choice, and skip choice.
- [x] Regression-test existing manual `money providers configure plaid`.
- [x] Regression-test existing Plaid Link flow remains unchanged.

### Phase 8: Docs

- [x] Update `README.md` command list.
- [x] Update `docs/GETTING_STARTED.md` setup walkthrough.
- [x] Update `docs/PRD.md` if the setup/login user-facing requirement changes.
- [x] Update `docs/CONFIG.md` for Plaid Dashboard auth storage beside the resolved config file.
- [x] Update `docs/ARCHITECTURE.md` to describe Plaid Dashboard login as a credential-bootstrap flow.
- [x] Update `docs/CONTRACTS.md` for new command JSON envelopes and errors.
- [x] Update `docs/DONORS.md` to record the Plaid CLI donor behavior and private-endpoint boundary.
- [x] Update `docs/ROADMAP.md` with the Plaid Dashboard Login milestone/status.
- [x] Check `IMPLEMENTATION_PLAN.md` for Plaid sections that need synchronization.

### Phase 9: Verification

- [x] Run `go test ./...` with writable Go caches if needed:

```bash
env GOCACHE=/private/tmp/money-go-build-2 GOPATH=/private/tmp/money-go go test ./...
```

- [x] Run manual help smoke tests:

```bash
go run ./cmd/money plaid login --help
go run ./cmd/money providers plaid login --help
go run ./cmd/money setup --help
```

- [x] Run a fake Dashboard login smoke test if the implementation exposes a local fake server.
- [x] Live Plaid Dashboard login was not run because it requires explicit user approval.

## User Decisions Needed Before Implementation

1. Approve use of Plaid CLI-compatible Dashboard OAuth and Dashboard API endpoints despite them being Plaid-specific and potentially private.
2. Confirm `sandbox` should be the default environment written by Dashboard login.
