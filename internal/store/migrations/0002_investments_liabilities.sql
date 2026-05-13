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
