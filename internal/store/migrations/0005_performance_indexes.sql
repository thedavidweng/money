-- Composite index for CashflowSummary: filters by removed, pending, currency, then range on date
CREATE INDEX IF NOT EXISTS idx_transactions_cashflow ON transactions (removed, pending, currency, date);

-- Composite index for filtered ListTransactions by account
CREATE INDEX IF NOT EXISTS idx_transactions_account_removed_date ON transactions (removed, account_id, date DESC);
