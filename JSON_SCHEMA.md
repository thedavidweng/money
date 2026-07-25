# JSON Envelope Schema

All commands emit a standard JSON envelope when invoked with `--json`.
The current schema version is `2026-07-25`.

Output is compact by default; pass the global `--pretty` flag for indented JSON.

## Success Envelope

```json
{
  "ok": true,
  "data": { ... },
  "meta": {
    "command": "transactions.list",
    "profile": "default",
    "duration_ms": 12,
    "schema_version": "2026-07-25",
    "request_id": "3f8a1c2e-9b4d-4e6a-8f21-0c5d7e2a1b9c",
    "pagination": {
      "limit": 50,
      "offset": 0,
      "has_more": false
    }
  }
}
```

### Fields

- `ok` — always `true` on success.
- `data` — command-specific payload. Collections are object-wrapped: `data.accounts`, `data.transactions`, `data.categories`, `data.tags`, `data.recurring`.
- `error` — omitted on success (see below).
- `meta` — request metadata (always present).
- `meta.command` — dot-separated command name.
- `meta.profile` — active configuration profile (`default` unless overridden).
- `meta.duration_ms` — wall-clock time spent handling the invocation, in milliseconds.
- `meta.schema_version` — envelope schema version as a date string.
- `meta.request_id` — UUID v4 generated once per invocation.
- `meta.demo` — `true` when running in demo mode (omitted otherwise).
- `meta.pagination` — present on list commands (`limit`, `offset`, `has_more`, `total` when available).
- `meta.warnings` — array of structured warning objects (`code`, `message`, `category`); omitted when empty.

## Error Envelope

```json
{
  "ok": false,
  "data": { ... },  // omitted when empty (omitempty)
  "error": {
    "code": "SYNC_PARTIAL_FAILURE",
    "message": "One or more provider items failed to sync",
    "category": "api",
    "retryable": true
  },
  "meta": {
    "command": "sync",
    "profile": "default",
    "duration_ms": 240,
    "schema_version": "2026-07-25",
    "request_id": "b1e6c7d4-2a3f-4c8b-9d10-6f2e5a7c3d81"
  }
}
```

### Error Fields

- `ok` — always `false` on error.
- `error` — a single structured error object. When several errors are aggregated (e.g. multi-item partial failures), the primary error is the object and the remaining errors are carried in `error.details`.
- Each error: `code`, `message`, `category`, `retryable`, and optional `details` (an array of the same shape).

## Error Taxonomy

Provider errors are classified as:

| Code | Category | Retryable | Description |
|------|----------|-----------|-------------|
| `missing_credentials` | auth | false | Provider credentials not configured |
| `invalid_authorization` | auth | false | Provider rejected credentials |
| `reconnect_required` | auth | false | Provider item needs re-linking |
| `rate_limit` | network | true | Provider rate limit hit |
| `network` | network | true | Network-level failure |
| `unsupported_feature` | api | false | Provider doesn't support this operation |
| `provider_api` | api | varies | Provider returned an error |
| `validation` | validation | false | Input validation failed |
| `internal` | internal | false | Internal error |

## Shared Error Codes

| Code | Category | Retryable | Exit Code | Meaning |
|------|----------|-----------|-----------|---------|
| `BASE_CONFIG_MISSING` | config | false | 3 | Config doesn't exist yet |
| `NOT_LOGGED_IN` | auth | false | 3 | Dashboard auth required |
| `TEAM_SELECTION_REQUIRED` | validation | false | 7 | Multiple teams, no selection |
| `API_KEYS_FETCH_REQUIRED` | auth | true | 3 | Dashboard auth exists but API keys need fetching |
| `DASHBOARD_TOKEN_REFRESH_FAILED` | auth | false | 3 | Refresh token expired |
| `READ_ONLY_VIOLATION` | safety | false | 4 | Mutation blocked by read-only mode |
| `CONFIRMATION_REQUIRED` | validation | false | 7 | JSON write without `--confirm` or `--dry-run` |
| `CONFIRMATION_REQUIRED` | safety | false | 10 | Destructive op without `--confirm` (via requireConfirm) |
| `SYNC_PARTIAL_FAILURE` | api | true | 6 | Some provider items failed |
| `CONFIG_WRITE_FAILED` | config | false | 1 | Config/env file write failure |
| `DB_BACKUP_FAILED` | safety | false | 1 | Pre-repair DB backup failure |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Internal or unclassified failure |
| 2 | Invalid command arguments |
| 3 | Authentication or provider authorization required |
| 4 | Read-only violation |
| 6 | Provider/API/schema/feature failure |
| 7 | Validation failure |
| 10 | Confirmation required |

## Schema Versioning

`schema_version` is a date string. It bumps to the date of the change on any
breaking change to the envelope shape; additive, backward-compatible fields do
not bump it. The `2026-07-25` version unified the envelope with the CLI fleet:
a single `error` object replaced the previous `errors[]` array, warnings moved
under `meta.warnings`, `meta` gained `profile`, `duration_ms`, and `request_id`,
and the redundant `meta.generated_at` was dropped.
