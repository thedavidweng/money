# Command Contracts

`money` commands write one JSON envelope to stdout when `--json` is set. Human mode may print compact text, but automation should use JSON.

## Envelope

All JSON commands use:

```json
{
  "ok": true,
  "data": {},
  "meta": {
    "command": "transactions.list",
    "schema_version": "0.1",
    "generated_at": "2026-05-10T00:00:00Z",
    "demo": false,
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

Errors use `ok: false` and `errors[]` entries with `code`, `message`, `category`, and `retryable`. JSON mode does not require stderr parsing.

## Read Commands

`accounts list --json` returns `data.accounts[]`.

Account records include local identity, display fields, balance as a decimal string, currency, and a `source` object. `source.kind` is `provider`, `manual`, or `import`; provider/import fields that do not apply are `null`.

`transactions list --json` and `transactions search <query> --json` return `data.transactions[]`.

Transaction records include local `id`, local `account_id`, `date`, optional `authorized_date`, decimal-string `amount`, `currency`, names, category fields, pending/review/removed flags, tags, note, recurring id, `last_changed_at`, and `source`.

`transactions list` supports:

- `--account`
- `--category`
- `--merchant`
- `--tag`
- `--date-from`
- `--date-to`
- `--needs-review true|false`
- `--pending true|false`
- `--recurring true|false`
- `--removed exclude|include|only`
- `--limit`
- `--offset`

Transactions sort by `date DESC`, then `pending DESC`, then `id ASC`. Accounts sort by hidden flag, type, name, and id.

`categories list --json`, `tags list --json`, and `recurring list --json` return `data.categories[]`, `data.tags[]`, and `data.recurring[]`.

## Sync Command

`sync --json` returns:

```json
{
  "items": [
    {
      "provider": "plaid",
      "provider_item_id": "pi_...",
      "status": "ok",
      "accounts_seen": 1,
      "transactions_added": 2,
      "transactions_modified": 0,
      "transactions_removed": 0,
      "recurring_streams_seen": 0,
      "next_transaction_cursor": "cursor"
    }
  ],
  "warnings": []
}
```

No linked Provider Items is a successful empty result with warning code `NO_LINKED_PROVIDER_ITEMS`.

Partial failures return `ok: false`, error code `SYNC_PARTIAL_FAILURE`, exit code `6`, and keep per-item diagnostics in `data.items[]`. Successful item results remain present.

`sync` supports:

- `--provider`
- `--provider-item`
- `--verbose`

Human mode defaults to a compact summary. `--verbose` prints per-item status.

## Error Taxonomy

Provider errors are classified as:

- `missing_credentials`
- `invalid_authorization`
- `reconnect_required`
- `rate_limit`
- `network`
- `unsupported_feature`
- `provider_api`
- `validation`
- `internal`

Reconnect-required provider states are machine-readable through item status or classified provider errors. Removed provider transactions are soft-deleted: normal reads exclude them, `--removed include|only` can show them, and permanent purge is reserved for an explicit confirmed cleanup command.

## Examples

```bash
money demo accounts list --json
money demo transactions list --json --merchant Coffee --pending true --limit 10
money demo transactions search coffee --json --limit 5
money sync --json
money sync --provider plaid --provider-item pi_example --json
```

## Monarch Compatibility Notes

The command names and stdout/stderr discipline follow Monarch CLI habits where useful. `money` differs by using object-wrapped collection fields, multi-error envelopes, explicit source provenance, encrypted local storage, and BYOK Provider adapters.
