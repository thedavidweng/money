# Fleet envelope core

Status: Accepted (amends ADR-0002)

## Context

`money` is one of five sibling CLIs (canvas, zenodo, flickr, monarch, money) that
all emit a machine-readable JSON envelope. Round 1 aligned the tools on a shared
core, but `money` still carried three shapes the rest of the fleet did not: a
top-level `errors[]` array, a top-level `warnings[]` array, and a `meta.generated_at`
timestamp. Divergent envelopes force agents and scripts to special-case each tool,
which is the opposite of what a fleet-wide contract is for.

ADR-0002 froze the previous shape (`{ ok, data, meta, warnings, errors }` with an
errors array "never a single object" and `meta` carrying `command`, `schema_version`,
`generated_at`). That decision predates the fleet-unification effort and its
envelope clauses are superseded here.

## Decision

Adopt the fleet core envelope. This is a breaking change; `schema_version` becomes
the date string `2026-07-25`.

- `errors[]` collapses to a single `error` object `{ code, message, category, retryable, details? }`. When multiple errors are aggregated (e.g. multi-item partial failures) the primary error is the object and the remainder go into `error.details`. `error` is omitted on success.
- Top-level `warnings[]` moves to `meta.warnings` (omitted when empty).
- `meta` gains `profile` (the active configuration profile), `duration_ms` (wall-clock handling time), and `request_id` (a UUID v4 generated once per invocation).
- `meta.generated_at` is dropped; it was redundant with request-scoped timing and non-deterministic in golden tests.

Tool-specific additive `meta` fields (`demo`, `pagination`) are retained. A single
`runtimeState.writeEnvelope` chokepoint stamps the request-scoped fields and honors
the global `--pretty` flag, so envelope bytes cannot drift between commands.

## Consequences

- Consumers that read `errors[0]` must read `error`; consumers that read top-level `warnings` must read `meta.warnings`. There is no compatibility shim.
- `request_id` gives each invocation a stable correlation id for logs and audits; `duration_ms` exposes handling latency without a wall-clock timestamp.
- `JSON_SCHEMA.md`, `docs/CONTRACTS.md`, and every inline/e2e envelope assertion were updated to the new shape.
- `github.com/google/uuid` is a direct dependency, matching the sibling tools.
