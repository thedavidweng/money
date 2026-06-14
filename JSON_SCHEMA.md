# JSON Envelope Schema

All commands emit a standard JSON envelope when invoked with `--json`.
The current schema version is `0.1`.

## Success Envelope

```json
{
  "ok": true,
  "data": { ... },
  "meta": {
    "command": "transactions.list",
    "schema_version": "0.1",
    "generated_at": "2026-05-10T00:00:00Z",
    "pagination": {
      "limit": 50,
      "offset": 0,
      "has_more": false
    }
  },
  "warnings": [],
  "errors": []
}
```

### Fields

- `ok` — always `true` on success.
- `data` — command-specific payload. Collections are object-wrapped: `data.accounts`, `data.transactions`, `data.categories`, `data.tags`, `data.recurring`.
- `meta` — request metadata (always present).
- `meta.command` — dot-separated command name.
- `meta.schema_version` — semver string.
- `meta.generated_at` — ISO 8601 timestamp.
- `meta.demo` — `true` when running in demo mode.
- `meta.pagination` — present on list commands (`limit`, `offset`, `has_more`, `total` when available).
- `warnings` — array of structured warning objects with `code`, `message`, `category`.
- `errors` — empty array on success.

## Error Envelope

```json
{
  "ok": false,
  "data": { ... },  // omitted when empty (omitempty)
  "meta": {
    "command": "sync",
    "schema_version": "0.1",
    "generated_at": "2026-05-10T00:00:00Z"
  },
  "warnings": [],
  "errors": [
    {
      "code": "SYNC_PARTIAL_FAILURE",
      "message": "One or more provider items failed to sync",
      "category": "api",
      "retryable": true
    }
  ]
}
```

### Error Fields

- `ok` — always `false` on error.
- `errors` — array of error objects (supports multi-error envelopes for partial failures).
- Each error: `code`, `message`, `category`, `retryable`.

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
| `TEAM_SELECTION_REQUIRED` | validation | false | 2 | Multiple teams, no selection |
| `API_KEYS_FETCH_REQUIRED` | auth | true | 3 | Dashboard auth exists but API keys need fetching |
| `DASHBOARD_TOKEN_REFRESH_FAILED` | auth | false | 3 | Refresh token expired |
| `READ_ONLY_VIOLATION` | safety | false | 4 | Mutation blocked by read-only mode |
| `CONFIRMATION_REQUIRED` | validation | false | 2 | JSON write without `--confirm` or `--dry-run` |
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

- **Major** — breaking contract changes.
- **Minor** — additive compatible fields.
- **Patch** — implementation-only changes (no schema change).
