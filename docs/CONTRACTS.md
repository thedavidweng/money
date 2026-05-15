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
- `--start-date` (backfill transactions from this date, YYYY-MM-DD; requires --end-date)
- `--end-date` (backfill transactions until this date, YYYY-MM-DD; requires --start-date)

Human mode defaults to a compact summary. `--verbose` prints per-item status.

## Investments, Liabilities, and Items Commands

These are read-only commands that query local data only.

### Investments

`investments holdings --json` returns `data.holdings[]`.

InvestmentHolding records include `id`, `account_id`, `security_id`, `quantity`, `institution_price`, `institution_value`, optional `cost_basis`, and `currency`.

`investments securities --json` returns `data.securities[]`.

InvestmentSecurity records include `id`, `security_id`, optional `isin`, `cusip`, `sedol`, `name`, optional `ticker_symbol`, `type`, `close_price`, optional `close_price_as_of`, and `currency`.

### Liabilities

`liabilities list --json` returns `data.liabilities[]`.

Liability records include `id`, `account_id`, `type`, `current_balance`, optional `original_balance`, `currency`, `name`, optional `last_payment_date`, optional `last_payment_amount`, optional `next_payment_due_date`, optional `apr`, and `updated_at`.

### Provider Items

`items list --json` returns `data.items[]`.

Provider item records include `id`, `provider`, `institution_id`, `alias`, `status`, `products`, and `transaction_cursor`.

`items get <id> --json` returns a single `data.item` object with the same fields.

`items rename <id> <name>` updates the alias for a provider item. No JSON contract output is produced in non-JSON mode; in JSON mode it returns `data.id` and `data.alias`.

`items remove <id>` removes a provider item and cascades delete all associated transactions, accounts, sync runs, recurring streams, and tags.

## Global Flags

All commands support:

- `--config` (config file path)
- `--profile` (configuration profile, default "default")
- `-j, --json` (write JSON envelope to stdout)

The `--profile` flag selects a named configuration profile. The default profile uses `~/.money/config.yaml`; custom profiles use `~/.money/profiles/<name>/config.yaml`.

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

Shared command error codes include:

| Code | Category | Retryable | Exit code | Meaning |
| --- | --- | --- | --- | --- |
| `BASE_CONFIG_MISSING` | `config` | false | 3 | `money plaid login` was run before the base config exists. |
| `NOT_LOGGED_IN` | `auth` | false | 3 | Dashboard-only operation needs Plaid Dashboard auth. |
| `TEAM_SELECTION_REQUIRED` | `validation` | false | 2 | Multiple Plaid Dashboard teams exist and no interactive or `--team` selection was available. |
| `API_KEYS_FETCH_REQUIRED` | `auth` | true | 3 | Dashboard auth exists but Plaid API keys still need to be fetched. |
| `DASHBOARD_TOKEN_REFRESH_FAILED` | `auth` | false | 3 | Stored Dashboard refresh token cannot refresh; rerun `money plaid login`. |
| `DASHBOARD_CONTRACT_CHANGED` | `api` | false | 6 | Plaid Dashboard private response shape no longer matches the known CLI-compatible contract. |
| `PLAID_DASHBOARD_LOGIN_REJECTED` | `auth` | false | 3 | Plaid rejected the CLI-compatible Dashboard OAuth path. |
| `PLAID_CREDENTIALS_OVERWRITE_REQUIRED` | `safety` | false | 10 | Existing Plaid API credentials would be replaced and `--force` is required. |
| `PLAID_ENVIRONMENT_NOT_PROVISIONED` | `validation` | false | 2 | Dashboard returned no secret for the selected Plaid environment. |
| `READ_ONLY_VIOLATION` | `safety` | false | 4 | The command would mutate local config, env, store, or auth files while read-only mode is enabled. |

## Examples

```bash
money setup --json
money doctor --json
money version --json
money providers configure plaid --client-id ... --secret ... --environment sandbox --json
money plaid login --json
money providers plaid login --json
money plaid logout --json
money demo accounts list --json
money demo transactions list --json --merchant Coffee --pending true --limit 10
money demo transactions search coffee --json --limit 5
money sync --json
money sync --provider plaid --provider-item pi_example --json
money sync --start-date 2024-01-01 --end-date 2024-03-01 --json
money providers plaid link --optional-products auth --additional-consented-products investments
money plaid sandbox link --products transactions --institution-id ins_56

# Read synced data
money investments holdings --json
money investments securities --json
money liabilities list --json
money items list --json
money items rename <id> "My Bank" --json
```

## Link Commands

Plaid link commands are human-mode browser flows. JSON mode is rejected for live Link because completion depends on a local callback.

`money link <institution-query>` and `money providers plaid link` support:

- `--no-open`
- `--additional-consented-products`
- `--required-if-supported-products`
- `--optional-products`

The three consent flags accept comma-separated Plaid product names and are validated before Plaid is called. `products` remains the configured initial product set. Canceling Plaid Link exits without exchanging a public token or writing a Provider Item. Plaid Link errors preserve Plaid error type/code/message plus request and link session metadata internally for diagnostics.

`money plaid sandbox link` creates a Plaid Sandbox public token, exchanges it through the same Provider Item persistence path as browser Link, and stores the linked item in the encrypted local store. It only runs when `providers.plaid.environment` is `sandbox`. `balance` is rejected as an initial product instead of being remapped.

## Setup Command

`setup --json` returns:

```json
{
  "config_path": "~/.money/config.yaml",
  "env_path": "~/.money/.env",
  "database_path": "~/.money/data/money.db",
  "secret_created": true,
  "db_created": true,
  "diagnostics": []
}
```

`setup` supports:

- `--force` (overwrite existing encryption key)
- `--profile` (use a named configuration profile)

## Doctor Command

`doctor --json` returns:

```json
{
  "diagnostics": [
    {
      "section": "Config",
      "code": "CONFIG_OK",
      "status": "ok",
      "message": "Config loaded from ~/.money/config.yaml",
      "category": "config"
    }
  ]
}
```

Diagnostic status values: `ok`, `warn`, `error`. Sections: `Config`, `Store`, `Providers`, `Links`, `Sync`, `Warnings`.

## Version Command

Default output is plain text: `money 0.1.0 (commit abc1234)`.

`version --json` returns:

```json
{
  "version": "0.1.0",
  "commit": "abc1234"
}
```

## Providers Configure Command

`providers configure plaid --json` returns:

```json
{
  "provider": "plaid",
  "keys_written": 2,
  "diagnostics": []
}
```

## Plaid Dashboard Login Commands

`plaid login --json` and `providers plaid login --json` return:

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

`plaid logout --json` and `providers plaid logout --json` return:

```json
{
  "provider": "plaid",
  "dashboard_auth_removed": true,
  "dashboard_auth_path": "/Users/you/.money/plaid-dashboard-auth.json",
  "api_keys_preserved": true,
  "env_path": "/Users/you/.money/.env"
}
```

These commands never include Plaid API secrets, Dashboard OAuth tokens, masked secrets, or reversible previews.

## Monarch Compatibility Notes

The command names and stdout/stderr discipline follow Monarch CLI habits where useful. `money` differs by using object-wrapped collection fields, multi-error envelopes, explicit source provenance, encrypted local storage, and BYOK Provider adapters.
