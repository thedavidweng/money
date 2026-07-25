# JSON envelope and exit-code contract

Status: Accepted (envelope shape superseded by ADR-0003; exit-code contract still current)

## Context

`money` is an agent-facing backend, so every command's machine output and process
exit code are a stable contract that scripts, agents, and cron jobs depend on. Two
drift risks exist: the JSON envelope shape can vary command to command, and the
documented exit-code table can disagree with the codes the binary actually emits.

`JSON_SCHEMA.md` already documents the envelope and a shared error/exit-code table.
`docs/ARCHITECTURE.md` fixes the error categories and exit-code meanings. Before this
record, several argument-validation paths (`CONFIRMATION_REQUIRED`,
`TEAM_SELECTION_REQUIRED`, interactive-link and provider-configure validation errors)
carried category `validation` but exited `2`, while the exit-code table assigns `7` to
validation failures and reserves `2` for invalid command-line arguments. That is a
code-vs-docs drift, not a contract the tools should keep.

## Decision

The success envelope is `{ ok, data, meta, warnings, errors }`. `meta` always carries
`command`, `schema_version`, and `generated_at`; `meta.demo` is present only in demo
mode; `meta.pagination` is present on paginated list commands. Collections are
object-wrapped under plural keys (`data.accounts`, `data.transactions`, ...). Errors use
an `errors` array (never a single object) so partial-sync and multi-diagnostic failures
are representable; each error carries `code`, `message`, `category`, `retryable`.
`WriteJSON` emits exactly one indented envelope to stdout. These field names are frozen;
the schema version bumps on breaking changes.

A single `render(stdout, state, command, data, table, opts...)` helper owns the
JSON-vs-human branch, the `meta.demo` wiring, and (via `withPagination`) the
`meta.pagination` wiring, so envelope bytes cannot drift between commands.

Exit codes follow the documented table: `0` success, `1` internal/unclassified, `2`
invalid command-line arguments, `3` auth/provider-authorization required, `4` read-only
violation, `6` provider/API/schema/feature failure, `7` validation failure, `10`
confirmation required. Category `validation` maps to exit `7`. All argument-validation
errors that previously exited `2` now exit `7`, matching their category. Exit `2` remains
reserved for CLI usage errors and is not emitted by any classified `validation` path.

## Consequences

- `CONFIRMATION_REQUIRED` (category `validation`) and `TEAM_SELECTION_REQUIRED` now exit
  `7`; the safety-category `CONFIRMATION_REQUIRED` destructive-gate path still exits `10`.
- `JSON_SCHEMA.md`'s shared error table was corrected to `7` for those validation rows.
- Existing envelope field names and the other exit codes are unchanged; e2e golden tests
  continue to guard the envelope bytes.
- New commands must route machine output through `render` and classify validation
  failures as category `validation` so they inherit exit `7` automatically.
