# Store Schema

This document is the first migration contract for the local SQLite store. The implementation may split the DDL across migration files, but the first schema must preserve these table names, columns, constraints, and indexes unless the architecture contract changes first.

## Migration Rules

- Migration files live under `internal/store/migrations`.
- File names use a monotonic numeric prefix: `0001_initial.sql`, `0002_name.sql`, and so on.
- Real stores are opened through the encrypted SQLite driver before migrations run.
- Demo mode applies the same migration SQL to an in-memory SQLite database, seeds bundled fixtures, and never opens or migrates the user's real database.
- Migrations must not create donor, AI, hosted billing, telemetry, or chat tables.
- Destructive migrations require a same-directory `.bak` copy of the database before the migration starts.

## ID and Money Conventions

- Local IDs are text primary keys with stable prefixes: `inst_`, `pi_`, `acc_`, `tx_`, `cat_`, `tag_`, `rec_`, `sec_`, `hold_`, `lia_`, `bud_`, `bcat_`, and `rule_`.
- Provider-native IDs are provenance fields, never local primary keys.
- Money is stored as signed integer minor units with an ISO 4217 `currency` column.
- JSON contracts render money as decimal strings such as `"123.45"` or `"-45.67"`.
- Date-only fields use `YYYY-MM-DD`. Datetime fields use UTC ISO 8601 strings.

## DDL

```sql
CREATE TABLE institutions (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  provider TEXT,
  provider_institution_id TEXT,
  url TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (provider, provider_institution_id)
);

CREATE TABLE provider_items (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  institution_id TEXT REFERENCES institutions(id),
  provider_external_item_id TEXT NOT NULL,
  encrypted_access_token BLOB NOT NULL,
  external_user_id TEXT,
  status TEXT NOT NULL,
  products_json TEXT NOT NULL DEFAULT '[]',
  transaction_cursor TEXT,
  last_successful_sync_at TEXT,
  reconnect_required_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (provider, provider_external_item_id)
);

CREATE TABLE accounts (
  id TEXT PRIMARY KEY,
  provider_item_id TEXT REFERENCES provider_items(id),
  institution_id TEXT REFERENCES institutions(id),
  provider_account_id TEXT,
  source_kind TEXT NOT NULL CHECK (source_kind IN ('provider', 'manual', 'import')),
  import_source_id TEXT,
  import_batch_id TEXT,
  name TEXT NOT NULL,
  official_name TEXT,
  alias TEXT,
  mask TEXT,
  type TEXT NOT NULL,
  subtype TEXT,
  current_balance_minor_units INTEGER NOT NULL,
  available_balance_minor_units INTEGER,
  available_credit_minor_units INTEGER,
  currency TEXT NOT NULL,
  hidden INTEGER NOT NULL DEFAULT 0 CHECK (hidden IN (0, 1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (provider_item_id, provider_account_id)
);

CREATE TABLE categories (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  group_name TEXT,
  hidden INTEGER NOT NULL DEFAULT 0 CHECK (hidden IN (0, 1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (group_name, name)
);

CREATE TABLE tags (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE transactions (
  id TEXT PRIMARY KEY,
  account_id TEXT NOT NULL REFERENCES accounts(id),
  provider_item_id TEXT REFERENCES provider_items(id),
  provider_transaction_id TEXT,
  provider_account_id TEXT,
  import_source_id TEXT,
  import_batch_id TEXT,
  source_row_hash TEXT,
  source_kind TEXT NOT NULL CHECK (source_kind IN ('provider', 'manual', 'import')),
  date TEXT NOT NULL,
  authorized_date TEXT,
  datetime TEXT,
  authorized_datetime TEXT,
  amount_minor_units INTEGER NOT NULL,
  currency TEXT NOT NULL,
  name TEXT NOT NULL,
  merchant_name TEXT,
  category_id TEXT REFERENCES categories(id),
  category_name TEXT,
  category_source TEXT NOT NULL CHECK (category_source IN ('local', 'provider', 'import', 'none')),
  provider_category TEXT,
  provider_subcategory TEXT,
  pending INTEGER NOT NULL DEFAULT 0 CHECK (pending IN (0, 1)),
  removed INTEGER NOT NULL DEFAULT 0 CHECK (removed IN (0, 1)),
  needs_review INTEGER NOT NULL DEFAULT 0 CHECK (needs_review IN (0, 1)),
  note TEXT,
  recurring_id TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  removed_at TEXT,
  UNIQUE (provider_item_id, provider_transaction_id),
  UNIQUE (import_source_id, import_batch_id, source_row_hash)
);

CREATE TABLE transaction_tags (
  transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  PRIMARY KEY (transaction_id, tag_id)
);

CREATE TABLE recurring (
  id TEXT PRIMARY KEY,
  provider_item_id TEXT REFERENCES provider_items(id),
  provider_recurring_id TEXT,
  account_id TEXT REFERENCES accounts(id),
  provider_account_id TEXT,
  source_kind TEXT NOT NULL CHECK (source_kind IN ('provider', 'manual', 'import')),
  merchant_name TEXT NOT NULL,
  average_amount_minor_units INTEGER NOT NULL,
  currency TEXT NOT NULL,
  frequency TEXT NOT NULL,
  next_date TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (provider_item_id, provider_recurring_id)
);

CREATE TABLE sync_runs (
  id TEXT PRIMARY KEY,
  provider TEXT,
  provider_item_id TEXT REFERENCES provider_items(id),
  started_at TEXT NOT NULL,
  finished_at TEXT,
  status TEXT NOT NULL,
  accounts_seen INTEGER NOT NULL DEFAULT 0,
  transactions_added INTEGER NOT NULL DEFAULT 0,
  transactions_modified INTEGER NOT NULL DEFAULT 0,
  transactions_removed INTEGER NOT NULL DEFAULT 0,
  recurring_seen INTEGER NOT NULL DEFAULT 0,
  error_code TEXT,
  error_message TEXT
);

CREATE INDEX idx_accounts_list ON accounts(hidden, type, name, id);
CREATE INDEX idx_transactions_list ON transactions(date DESC, pending DESC, id ASC);
CREATE INDEX idx_transactions_account_date ON transactions(account_id, date DESC);
CREATE INDEX idx_transactions_category ON transactions(category_id);
CREATE INDEX idx_transactions_merchant ON transactions(merchant_name);
CREATE INDEX idx_transactions_review ON transactions(needs_review);
CREATE INDEX idx_transactions_removed ON transactions(removed);
CREATE INDEX idx_transaction_tags_tag ON transaction_tags(tag_id, transaction_id);
CREATE INDEX idx_recurring_account ON recurring(account_id);
CREATE INDEX idx_sync_runs_item_started ON sync_runs(provider_item_id, started_at DESC);
```

### Migration 0002: Investments and Liabilities

```sql
CREATE TABLE securities (
  id TEXT PRIMARY KEY,
  security_id TEXT NOT NULL UNIQUE,
  isin TEXT,
  cusip TEXT,
  sedol TEXT,
  name TEXT NOT NULL,
  ticker_symbol TEXT,
  type TEXT,
  close_price REAL,
  close_price_as_of TEXT,
  currency TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE holdings (
  id TEXT PRIMARY KEY,
  provider_item_id TEXT NOT NULL REFERENCES provider_items(id) ON DELETE CASCADE,
  account_id TEXT NOT NULL REFERENCES accounts(id),
  provider_account_id TEXT NOT NULL,
  security_id TEXT,
  quantity REAL NOT NULL,
  institution_price REAL NOT NULL,
  institution_value REAL NOT NULL,
  cost_basis REAL,
  currency TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (provider_item_id, provider_account_id, security_id)
);

CREATE TABLE liabilities (
  id TEXT PRIMARY KEY,
  provider_item_id TEXT NOT NULL REFERENCES provider_items(id) ON DELETE CASCADE,
  account_id TEXT NOT NULL REFERENCES accounts(id),
  provider_account_id TEXT NOT NULL,
  type TEXT NOT NULL,
  current_balance REAL NOT NULL,
  original_balance REAL,
  currency TEXT NOT NULL,
  name TEXT NOT NULL,
  last_payment_date TEXT,
  last_payment_amount REAL,
  next_payment_due_date TEXT,
  apr REAL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (provider_item_id, provider_account_id, type, name)
);

CREATE INDEX idx_holdings_account ON holdings(account_id);
CREATE INDEX idx_liabilities_account ON liabilities(account_id);
```

### Migration 0003: Budgets

```sql
CREATE TABLE budgets (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  currency TEXT NOT NULL,
  period TEXT NOT NULL CHECK (period IN ('monthly', 'yearly')),
  start_date TEXT NOT NULL,
  end_date TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE budget_categories (
  id TEXT PRIMARY KEY,
  budget_id TEXT NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
  category_id TEXT REFERENCES categories(id),
  name TEXT NOT NULL,
  limit_minor_units INTEGER NOT NULL,
  currency TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX idx_budgets_period ON budgets(period, start_date, end_date);
CREATE INDEX idx_budget_categories_budget ON budget_categories(budget_id);
```

### Migration 0004: Rules

```sql
CREATE TABLE rules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  condition_field TEXT NOT NULL,
  condition_op TEXT NOT NULL,
  condition_value TEXT NOT NULL,
  action_type TEXT NOT NULL,
  action_value TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX idx_rules_priority ON rules(enabled, priority DESC);
```

## Source Mapping

The stable JSON `source` object is reconstructed from table columns:

- `kind`: `source_kind`
- `provider`: `provider_items.provider`
- `provider_item_id`: `provider_item_id`
- `provider_external_item_id`: `provider_items.provider_external_item_id`
- `institution_id`: account or provider item institution
- `provider_account_id`: account or transaction provider account ID
- `provider_transaction_id`: transaction provider transaction ID
- `import_source_id`: import source ID
- `import_batch_id`: import batch ID

Every source field is present in JSON. Non-applicable values render as `null`.
