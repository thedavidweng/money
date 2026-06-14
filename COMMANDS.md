# Command Reference

All commands accept the [global flags](#global-flags) listed at the bottom.
Use `money <command> --help` for full flag details.

## Top-level Commands

| Command | Description |
|---------|-------------|
| `version` | Show version and commit |
| `doctor` | Check configuration, store, providers, and connectivity |
| `setup` | Guided first-time setup (config, encryption key, database) |
| `completion` | Generate shell completion scripts (bash, zsh, fish, powershell) |

```bash
money version
money version --json
money doctor
money doctor --json
money setup
money setup --json
```

## auth / plaid

Manage Plaid credentials and dashboard authentication.

| Command | Usage | Description |
|---------|-------|-------------|
| `plaid login` | `money plaid login` | Interactive Plaid Dashboard OAuth login |
| `plaid logout` | `money plaid logout` | Remove Dashboard auth (preserves API keys) |
| `providers configure plaid` | `money providers configure plaid --client-id ... --secret ...` | Set Plaid API keys directly |
| `providers plaid login` | `money providers plaid login` | Alias for `plaid login` |

```bash
money plaid login --json
money plaid logout --json
money providers configure plaid --client-id CLIENT_ID --secret SECRET --environment sandbox --json
```

## link

Link a financial institution via Plaid Link (browser flow).

| Command | Usage | Description |
|---------|-------|-------------|
| `link` | `money link "Chase"` | Open Plaid Link for an institution query |
| `plaid sandbox link` | `money plaid sandbox link --products transactions --institution-id ins_56` | Create a sandbox link (no browser) |

**Key flags:**

- `--no-open` — don't auto-open browser
- `--additional-consented-products` — extra Plaid products (comma-separated)
- `--required-if-supported-products` — required products if supported
- `--optional-products` — optional products

```bash
money link "Chase" --json
money plaid sandbox link --products transactions --institution-id ins_56
```

## accounts

Read local account data.

| Command | Usage | Description |
|---------|-------|-------------|
| `accounts list` | `money accounts list` | List all financial accounts |

```bash
money accounts list --json
```

## accounts create-manual

Create a local manual account (not linked to a provider).

```bash
money accounts create-manual --name Savings --type depository --balance 5000.00 --currency USD --confirm
money accounts create-manual --name "Credit Card" --type credit --balance 500.00 --dry-run
```

## transactions

Read and manage local transaction data.

| Command | Usage | Description |
|---------|-------|-------------|
| `transactions list` | `money transactions list` | List transactions with filters |
| `transactions search` | `money transactions search <query>` | Search transactions by text |

**Filters for `transactions list`:**

- `--account` — filter by account
- `--category` — filter by category
- `--merchant` — filter by merchant
- `--tag` — filter by tag
- `--date-from` / `--date-to` — date range
- `--needs-review true|false` — review status
- `--pending true|false` — pending status
- `--recurring true|false` — recurring status
- `--removed exclude|include|only` — removed records
- `--limit` / `--offset` — pagination

Sort order: `date DESC`, `pending DESC`, `id ASC`.

```bash
money transactions list --json
money transactions list --account acc_xxx --date-from 2024-01-01 --json
money transactions search "coffee" --json
```

## categories / tags / recurring

Read local category, tag, and recurring transaction data.

```bash
money categories list --json
money tags list --json
money recurring list --json
```

## sync

Refresh data from linked financial providers.

| Command | Usage | Description |
|---------|-------|-------------|
| `sync` | `money sync` | Sync all linked provider items |
| `sync` | `money sync --provider plaid` | Sync one provider |
| `sync` | `money sync --provider-item pi_xxx` | Sync one linked item |

**Key flags:**

- `--provider` — narrow to one provider
- `--provider-item` — narrow to one linked item
- `--verbose` — per-item status in human mode
- `--start-date` / `--end-date` — backfill date range (YYYY-MM-DD)

```bash
money sync --json
money sync --provider plaid --json
money sync --start-date 2024-01-01 --end-date 2024-03-01 --json
```

## investments

Read local investment data.

```bash
money investments holdings --json
money investments securities --json
```

## liabilities

Read local liability data.

```bash
money liabilities list --json
```

## items

Manage linked provider items.

| Command | Usage | Description |
|---------|-------|-------------|
| `items list` | `money items list` | List linked provider items |
| `items get` | `money items get <id>` | Show one provider item |
| `items rename` | `money items rename <id> "My Bank"` | Update alias |
| `items remove` | `money items remove <id>` | Remove item and cascade-delete associated data |

```bash
money items list --json
money items get pi_xxx --json
money items rename pi_xxx "My Bank" --json
money items remove pi_xxx --json
```

## demo

Run against a non-persistent demo environment with synthetic data.

```bash
money demo accounts list --json
money demo transactions list --json --merchant Coffee --pending true --limit 10
money demo transactions search coffee --json --limit 5
```

## cashflow

Show income and expenses over time.

```bash
money cashflow --from 2024-01-01 --to 2024-12-31
money cashflow --from 2024-01-01 --to 2024-12-31 --period yearly --json
```

## net-worth

Show current net worth across all visible accounts.

```bash
money net-worth
money net-worth --json
```

## budgets

Manage budgets.

```bash
money budgets list --json
money budgets create --name Groceries --period monthly --start-date 2024-01-01 --end-date 2024-12-31 --confirm
money budgets get <id> --json
money budgets delete <id>
```

## rules

Manage auto-categorization rules.

```bash
money rules list --json
money rules create --name "Mark Uber" --condition-field merchant_name --condition-op contains --condition-value uber --action-type set_category --action-value transport --confirm
money rules apply
money rules delete <id>
```

## budgets categories

Manage budget categories.

```bash
money budgets categories create --budget-id <id> --name Groceries --limit 50000 --confirm
money budgets categories delete <id>
```

## import

Import accounts and transactions from external sources.

```bash
money import monarch transactions.csv
money import csv transactions.csv --dry-run
money import monarch data.csv --batch-id 20240101 --confirm
```

## feedback

Submit feedback.

```bash
money feedback
```

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | | Config file path |
| `--profile` | `default` | Configuration profile |
| `-j, --json` | `false` | JSON envelope to stdout |
