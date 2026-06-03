# Release Plan: v0.1.0

This plan closes the gap between the current codebase and a first public release. It assumes `IMPLEMENTATION_PLAN.md` Phases 1–7 are complete (they are) and focuses only on the remaining work that blocks a tagged, installable, agent-usable release.

## Release Definition

A release is ready when all of the following hold:

- Every command listed as an "Initial command" in `docs/PRD.md` and `IMPLEMENTATION_PLAN.md` is implemented and covered by contract tests.
- A new user with no prior `~/.money` can run a single bootstrap command to reach a working encrypted store.
- `money doctor` returns structured diagnostics for config, store, providers, links, and sync readiness.
- `money version` obeys the `--json` flag and reports the real build version injected at link time.
- `go test ./...` and `go vet ./...` run in CI on every push and pull request.
- A tagged release (`v0.1.0`) exists on GitHub with prebuilt binaries for the supported platforms and a matching Go module proxy entry, so `go install github.com/thedavidweng/money/cmd/money@latest` resolves to that tag.
- README, `docs/PRD.md`, `docs/CONTRACTS.md`, and `docs/CONFIG.md` describe exactly what ships; no documented command or flag is missing from the binary.
- No runtime code references donor AI, Ray hosted proxy, subscription billing, or chat concepts (`rg` verification passes).

## Required Reading Before Implementation

1. `AGENTS.md`
2. `docs/PRD.md`
3. `docs/CONFIG.md`
4. `docs/CONTRACTS.md`
5. `docs/SCHEMA.md`
6. `IMPLEMENTATION_PLAN.md`

## Principles

- Occam's razor: build the minimum surface that closes each release criterion. Do not add speculative commands, flags, or configuration.
- No hidden fallbacks: every missing credential, missing file, or unopenable store must produce a stable error code.
- Small deep modules: keep onboarding helpers inside `internal/config` and `internal/cli`; do not introduce a new top-level package unless two or more callers need it.
- Dead-code sweep: after each phase, remove code paths that were only used by now-replaced scaffolding.

## Phase R1: Onboarding Commands

Goal: a new user can go from zero to a working encrypted store and diagnose problems without reading source.

### R1.1 `money setup`

Steps:

1. Add `money setup` command. Human mode runs an arrow-key interactive flow using `charm.land/huh/v2`; JSON mode requires all inputs via flags and fails with `VALIDATION` if required flags are missing.
2. Resolve config path with the same rules as `internal/config` load order; create `~/.money/` when using the default path.
3. Write a `config.yaml` skeleton with `database.path`, `database.encryption_key: {env: MONEY_DB_ENCRYPTION_KEY}`, and empty `providers:` block.
4. Generate a 32-byte random `MONEY_DB_ENCRYPTION_KEY`, base64url-encoded without padding, and write it to the resolved `.env` file with `0600` permissions on POSIX platforms.
5. Open the encrypted store and run migrations so that read commands work immediately after setup.
6. Never print the encryption key. Setup output reports file paths and boolean `secret_created: true`.
7. Support `--config <path>` to target an alternate location. Support `--force` in JSON mode to overwrite an existing env var value; human mode asks with the arrow-key selector.
8. On any write failure after partial writes, stop, report what was written and what failed, return exit code `1` (`INTERNAL` or `CONFIG_WRITE_FAILED`).
9. On DB open/migration failure after config/env writes, keep the written files, report the DB failure, and suggest running `money doctor`.
10. After success, run the shared doctor checks (see R1.2) and include their results in setup output.

Acceptance:

- First-time `money setup` on an empty home directory produces a working config, env, and encrypted DB.
- `money accounts list --json` succeeds with an empty `accounts` array immediately after setup.
- Re-running `money setup` on an already-configured install is a no-op for files that already exist and does not rotate the encryption key.
- Setup never prints secret values. JSON output exposes only booleans and paths for secrets.
- Setup does not attempt DB repair or re-encryption.

### R1.2 `money doctor`

Steps:

1. Add `money doctor` with sections `Config`, `Store`, `Providers`, `Links`, `Sync`, and `Warnings`. Each diagnostic is a `{ code, status, message, category }` record with status `ok`, `warn`, or `error`.
2. Implement config checks: config file exists, env file exists, required `database.path` and `database.encryption_key` resolvable, optional provider credentials resolvable.
3. Implement store checks: DB file exists at configured path, opens with the configured key, schema version matches latest migration.
4. Implement provider checks: for each registered provider, report configured vs available state (credentials present but valid/invalid distinction is `warn`-level only until sync is run).
5. Implement links check: count linked provider items per provider from the store.
6. Implement sync check: last `sync_runs` entry status and time; no runs is `warn` only if at least one provider item exists.
7. Implement warnings check: broad env-file permissions on POSIX, direct secret scalars in YAML, config pointing at non-default path without `--config` or `MONEY_CONFIG`.
8. Implement `money doctor --json`: return an envelope with `data.diagnostics[]` and `meta.command: "doctor"`. Do not include secret values.
9. Implement `money doctor --fix` with exactly these repairs: create missing `~/.money/`, create missing config skeleton, create missing env skeleton, generate missing DB encryption key, chmod env file to `0600` on POSIX.
10. Run the same diagnostic code from `money setup` so setup summary and doctor never diverge.

Acceptance:

- `money doctor` on an unconfigured system reports `Config: error` and exits non-zero.
- `money doctor --json` returns a parseable envelope even when config is missing.
- `money doctor --fix` is idempotent, never rotates keys, never edits user-set values, and never migrates donor config paths.
- `money doctor --fix --dry-run` prints the repair plan without writing.
- No diagnostic prints secret values or reversible partial previews.

### R1.3 `money providers configure <provider>`

Steps:

1. Add `money providers configure plaid` and `money providers configure bridge` commands.
2. Human mode uses the arrow-key selector for overwrite confirmation and free-text prompts for scalar credentials, displaying mask characters while typing secrets.
3. JSON mode requires all credential flags; partial credentials return `VALIDATION` and write nothing.
4. Write secrets to the resolved env file and corresponding `env:` references to `config.yaml`. Never write raw secret values into `config.yaml`.
5. Existing env vars are not overwritten unless human user confirms via the arrow-key selector or JSON caller passes `--force`.
6. After writing, run only that provider's doctor diagnostics plus global blocking diagnostics.
7. Configuration is atomic per provider. If any required field is missing or any write fails, stop immediately, report what was written, and return `CONFIG_WRITE_FAILED`.

Acceptance:

- `money providers configure plaid --client-id ... --secret ... --environment sandbox --json` writes env entries and YAML `env:` references, then prints Plaid-only diagnostics.
- Re-running with the same values is a no-op.
- Missing any required Plaid or Bridge field fails atomically before any file write.

## Phase R2: Version and Output Correctness

Goal: every command obeys the contract for stdout, `--json`, and reports the real build version.

### R2.1 Fix `money version`

Steps:

1. Change the default mode: when `--json` is not set, print a single plain-text line such as `money 0.1.0 (commit abc1234)`.
2. When `--json` is set, print the existing envelope.
3. Replace the hardcoded `"0.0.0"` with a package-level variable populated via `-ldflags "-X github.com/thedavidweng/money/internal/cli.Version=... -X github.com/thedavidweng/money/internal/cli.Commit=..."`.
4. `go build` without ldflags yields `dev` for both fields, never a stale semver.

Acceptance:

- `money version` prints plain text on stdout.
- `money version --json` prints the JSON envelope on stdout.
- A binary built with ldflags reports its tag and commit.
- Contract test covers both modes.

### R2.2 Table rendering for human mode

Steps:

1. Add `github.com/olekukonko/tablewriter` to `go.mod`.
2. Render `accounts list` human mode with columns `NAME`, `TYPE`, `BALANCE`, `AVAILABLE`, `CURRENCY`, `SOURCE`, `UPDATED`. Add `--verbose` for provider IDs and deeper provenance.
3. Render `transactions list` / `transactions search` human mode with columns `DATE`, `ACCOUNT`, `MERCHANT`, `AMOUNT`, `CATEGORY`, `STATUS`. Add `--verbose` for local IDs, provider category, tags, note, and source.
4. Render monetary values with explicit text signs; color is auxiliary only (green for positive, red for negative when stdout is a TTY).
5. JSON output must remain byte-equivalent to the current contract. Contract tests stay unchanged.

Acceptance:

- Human `accounts list` and `transactions list|search` produce aligned tables.
- Non-TTY stdout suppresses color escapes.
- JSON envelopes are unchanged.

### R2.3 Dead code sweep

Steps:

1. Run `go vet ./...` and address any warnings.
2. Review `internal/cli`, `internal/linking`, and `internal/providers` for helpers that have no callers after R1 lands.
3. Remove comments, fixtures, and branches that only existed to scaffold Phase 1–3 work.

## Phase R3: Release Infrastructure

Goal: every push is tested, and a tag produces published binaries.

### R3.1 CI workflow

Steps:

1. Create `.github/workflows/ci.yml` that triggers on `push` to `main` and on `pull_request`.
2. Steps: checkout, setup Go 1.26, `go vet ./...`, `go test ./... -race`, verify `go build ./...`.
3. Cache Go modules and the Go build cache keyed on `go.sum`.
4. Run on `ubuntu-latest` and `macos-latest`.
5. Make the README CI badge resolve to this workflow.

Acceptance:

- CI badge is green on `main`.
- A pull request that breaks a test fails CI.

### R3.2 Release workflow

Steps:

1. Add a `.goreleaser.yaml` configured for Linux amd64/arm64 and macOS amd64/arm64 binaries, archived as tarballs.
2. Inject `Version` and `Commit` via `-ldflags`.
3. Skip Docker images and Homebrew taps for v0.1.0; add them only if there is a real maintainer commitment.
4. Add `.github/workflows/release.yml` triggered on tags matching `v*`.
5. Release workflow: checkout with full history, setup Go, run `go test ./...`, run `goreleaser release --clean`.
6. Generate SHA256 checksums for every archive and include them in the GitHub release.

Acceptance:

- Pushing a tag `v0.1.0-rc.1` produces a draft GitHub release with platform archives and checksums.
- Downloaded binaries report the tag version via `money version`.

### R3.3 Module and tag

Steps:

1. Confirm the Go module path matches the repository path and that no replace directives leak into the public module.
2. Tag `v0.1.0` after R1–R3.2 land and CI is green on `main`.
3. Verify `go install github.com/thedavidweng/money/cmd/money@v0.1.0` succeeds on a clean machine.
4. Update the README install instructions to reference `@latest` and to mention the release page for prebuilt binaries.

Acceptance:

- `go install github.com/thedavidweng/money/cmd/money@latest` installs a binary that prints `money v0.1.0` on `version`.
- The release page contains archives for the four target platforms plus checksums.

## Phase R4: Documentation Alignment

Goal: every documented command and flag exists in the binary, and every binary command is documented.

Steps:

1. Walk `docs/PRD.md` "Initial commands" list; mark each as implemented or remove it from the initial list.
2. Update `docs/CONTRACTS.md` to cover `setup`, `doctor`, `version`, and `providers configure` JSON shapes.
3. Update `README.md` command list to exactly match the binary.
4. Remove any deferred capability from user-facing docs that is not behind a flag or subcommand in v0.1.0.
5. Verify `docs/ROADMAP.md` Phase 1 and Phase 2 descriptions match what shipped.

Acceptance:

- Running `money --help` and comparing against the README command list shows no missing or extra commands.
- Every `--json` command named in `docs/CONTRACTS.md` returns a valid envelope in a contract test.

## Verification Before Tagging

Run all of the following on a clean workspace. All must pass:

```bash
go vet ./...
go test ./... -race
go run ./cmd/money version --json
go run ./cmd/money setup --config /tmp/money/config.yaml --json
go run ./cmd/money --config /tmp/money/config.yaml doctor --json
go run ./cmd/money --config /tmp/money/config.yaml accounts list --json
rg -n "RAY_API_KEY|RAY_PROXY_BASE|rayfinance\.app|Anthropic|OpenAI|conversation_history|ai_audit_log|memories" --glob '!donors/**'
git status --short --ignored
```

The `rg` search must return zero matches outside `donors/`. `git status` must be clean.

## Out of Scope for v0.1.0

These are deferred to v0.2.0 or later and must not block the tag:

- `money accounts update <id> --alias ...`
- `money transactions cleanup --removed ...`
- Monarch, CSV, and Apple Card import sources
- MX and Finicity provider adapters
- Budgeting, rules, cashflow, net worth primitives
- Homebrew tap, Docker image, Windows builds

If any of these reappear in implementation work before v0.1.0, move them back to `IMPLEMENTATION_PLAN.md` under a new phase instead of expanding this release plan.
