package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/thedavidweng/money/internal/core"
)

func (s *SQLiteStore) UpsertAccount(ctx context.Context, account core.FinancialAccount) error {
	now := time.Now().UTC().Format(time.RFC3339)
	id, err := s.localAccountIDForProviderAccount(ctx, account.ProviderItemID, account.ProviderAccountID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO accounts (
  id, provider_item_id, provider_account_id, source_kind, name, official_name, mask,
  type, subtype, current_balance_minor_units, available_balance_minor_units,
  available_credit_minor_units, currency, created_at, updated_at
) VALUES (?, ?, ?, 'provider', ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider_item_id, provider_account_id) DO UPDATE SET
  name = excluded.name,
  official_name = excluded.official_name,
  mask = excluded.mask,
  type = excluded.type,
  subtype = excluded.subtype,
  current_balance_minor_units = excluded.current_balance_minor_units,
  available_balance_minor_units = excluded.available_balance_minor_units,
  available_credit_minor_units = excluded.available_credit_minor_units,
  currency = excluded.currency,
  updated_at = excluded.updated_at`,
		id, account.ProviderItemID, account.ProviderAccountID, account.Name, account.OfficialName, account.Mask,
		account.Type, account.Subtype, account.CurrentBalanceMinorUnits, account.AvailableBalanceMinorUnits,
		account.AvailableCreditMinorUnits, account.Currency, now, now)
	return err
}

// sqlExecer is the common interface satisfied by both *sql.DB and *sql.Tx.
type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *SQLiteStore) UpsertTransaction(ctx context.Context, transaction core.ProviderTransaction) error {
	return upsertTransactionExec(ctx, s.db, transaction)
}

// UpsertTransactions inserts or updates multiple transactions in a single SQL transaction.
func (s *SQLiteStore) UpsertTransactions(ctx context.Context, transactions []core.ProviderTransaction) error {
	if len(transactions) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, txn := range transactions {
		if err := upsertTransactionExec(ctx, tx, txn); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// upsertTransactionExec contains the core upsert logic shared by UpsertTransaction and UpsertTransactions.
func upsertTransactionExec(ctx context.Context, exec sqlExecer, transaction core.ProviderTransaction) error {
	now := time.Now().UTC().Format(time.RFC3339)
	id, err := localTransactionIDForProviderTransaction(ctx, exec, transaction.ProviderItemID, transaction.ProviderTransactionID)
	if err != nil {
		return err
	}
	accountID, err := existingAccountIDForProviderAccount(ctx, exec, transaction.ProviderItemID, transaction.ProviderAccountID)
	if err != nil {
		return err
	}
	categorySource := "none"
	var categoryName sql.NullString
	if transaction.ProviderCategory != nil && *transaction.ProviderCategory != "" {
		categorySource = "provider"
		categoryName = sql.NullString{String: *transaction.ProviderCategory, Valid: true}
	}
	_, err = exec.ExecContext(ctx, `
INSERT INTO transactions (
  id, account_id, provider_item_id, provider_transaction_id, provider_account_id,
  source_kind, date, authorized_date, amount_minor_units, currency, name, merchant_name,
  category_name, category_source, provider_category, provider_subcategory,
  pending, removed, needs_review, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, 'provider', ?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?, 0, 0, ?, ?)
ON CONFLICT(provider_item_id, provider_transaction_id) DO UPDATE SET
  account_id = excluded.account_id,
  provider_account_id = excluded.provider_account_id,
  date = excluded.date,
  authorized_date = excluded.authorized_date,
  amount_minor_units = excluded.amount_minor_units,
  currency = excluded.currency,
  name = excluded.name,
  merchant_name = excluded.merchant_name,
  category_name = excluded.category_name,
  category_source = excluded.category_source,
  provider_category = excluded.provider_category,
  provider_subcategory = excluded.provider_subcategory,
  pending = excluded.pending,
  removed = 0,
  updated_at = excluded.updated_at`,
		id, accountID, transaction.ProviderItemID, transaction.ProviderTransactionID, transaction.ProviderAccountID,
		transaction.Date, ptrNullableString(transaction.AuthorizedDate), transaction.AmountMinorUnits, transaction.Currency,
		transaction.Name, transaction.MerchantName, categoryName, categorySource, ptrNullableString(transaction.ProviderCategory),
		ptrNullableString(transaction.ProviderSubcategory), boolInt(transaction.Pending), now, now)
	return err
}

func (s *SQLiteStore) UpsertRecurring(ctx context.Context, recurring core.ProviderRecurring) error {
	now := time.Now().UTC().Format(time.RFC3339)
	id, err := s.localRecurringIDForProviderRecurring(ctx, recurring.ProviderItemID, recurring.ProviderRecurringID)
	if err != nil {
		return err
	}
	accountID, err := s.existingAccountIDForProviderAccount(ctx, recurring.ProviderItemID, recurring.ProviderAccountID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO recurring (
  id, provider_item_id, provider_recurring_id, account_id, provider_account_id,
  source_kind, merchant_name, average_amount_minor_units, currency, frequency,
  next_date, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, 'provider', ?, ?, ?, ?, ?, 'active', ?, ?)
ON CONFLICT(provider_item_id, provider_recurring_id) DO UPDATE SET
  account_id = excluded.account_id,
  provider_account_id = excluded.provider_account_id,
  merchant_name = excluded.merchant_name,
  average_amount_minor_units = excluded.average_amount_minor_units,
  currency = excluded.currency,
  frequency = excluded.frequency,
  next_date = excluded.next_date,
  status = excluded.status,
  updated_at = excluded.updated_at`,
		id, recurring.ProviderItemID, recurring.ProviderRecurringID, accountID, recurring.ProviderAccountID,
		recurring.MerchantName, recurring.AverageAmountMinorUnits, recurring.Currency, recurring.Frequency,
		ptrNullableString(recurring.NextDate), now, now)
	return err
}

func (s *SQLiteStore) MarkTransactionRemoved(ctx context.Context, providerItemID string, providerTransactionID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
UPDATE transactions
SET removed = 1, removed_at = ?, updated_at = ?
WHERE provider_item_id = ? AND provider_transaction_id = ?`,
		now, now, providerItemID, providerTransactionID)
	return err
}

func (s *SQLiteStore) RecordSyncRun(ctx context.Context, run core.SyncRun) error {
	id, err := core.NewLocalID("sync_")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO sync_runs (
  id, provider, provider_item_id, started_at, finished_at, status,
  accounts_seen, transactions_added, transactions_modified, transactions_removed, recurring_seen,
  error_code, error_message
)
VALUES (?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`,
		id, run.Provider, run.ProviderItemID, run.StartedAt, run.FinishedAt, run.Status,
		run.AccountsSeen, run.TransactionsAdded, run.TransactionsModified, run.TransactionsRemoved, run.RecurringStreamsSeen,
		run.ErrorCode, run.ErrorMessage)
	return err
}

// SyncRunSummary holds the latest sync run per provider item for doctor diagnostics.
type SyncRunSummary struct {
	ProviderItemID string
	Provider       string
	StartedAt      string
	FinishedAt     string
	Status         string
	ErrorCode      string
	ErrorMessage   string
}

// MarkStuckSyncRunsInterrupted marks sync runs with no finished_at as "interrupted".
// Returns the number of runs updated.
func (s *SQLiteStore) MarkStuckSyncRunsInterrupted(ctx context.Context) (int, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
UPDATE sync_runs
SET status = 'interrupted', finished_at = ?
WHERE finished_at IS NULL`, now)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// LatestSyncRuns returns the most recent sync run for each provider item.
func (s *SQLiteStore) LatestSyncRuns(ctx context.Context) ([]SyncRunSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT sr.provider_item_id, COALESCE(sr.provider, ''), sr.started_at, COALESCE(sr.finished_at, ''),
       sr.status, COALESCE(sr.error_code, ''), COALESCE(sr.error_message, '')
FROM sync_runs sr
INNER JOIN (
  SELECT provider_item_id, MAX(started_at) AS max_started
  FROM sync_runs
  GROUP BY provider_item_id
) latest ON sr.provider_item_id = latest.provider_item_id AND sr.started_at = latest.max_started
ORDER BY sr.provider_item_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var runs []SyncRunSummary
	for rows.Next() {
		var r SyncRunSummary
		if err := rows.Scan(&r.ProviderItemID, &r.Provider, &r.StartedAt, &r.FinishedAt, &r.Status, &r.ErrorCode, &r.ErrorMessage); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

func (s *SQLiteStore) localAccountIDForProviderAccount(ctx context.Context, providerItemID string, providerAccountID string) (string, error) {
	id, err := s.queryAccountIDForProviderAccount(ctx, providerItemID, providerAccountID)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	return core.NewLocalID("acc_")
}

func (s *SQLiteStore) existingAccountIDForProviderAccount(ctx context.Context, providerItemID string, providerAccountID string) (string, error) {
	return existingAccountIDForProviderAccount(ctx, s.db, providerItemID, providerAccountID)
}

func existingAccountIDForProviderAccount(ctx context.Context, exec sqlExecer, providerItemID string, providerAccountID string) (string, error) {
	id, err := queryAccountIDForProviderAccount(ctx, exec, providerItemID, providerAccountID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("provider account %q for provider item %q must be synced before dependent records", providerAccountID, providerItemID)
	}
	return id, err
}

func (s *SQLiteStore) queryAccountIDForProviderAccount(ctx context.Context, providerItemID string, providerAccountID string) (string, error) {
	return queryAccountIDForProviderAccount(ctx, s.db, providerItemID, providerAccountID)
}

func queryAccountIDForProviderAccount(ctx context.Context, exec sqlExecer, providerItemID string, providerAccountID string) (string, error) {
	var id string
	err := exec.QueryRowContext(ctx, `
SELECT id
FROM accounts
WHERE provider_item_id = ? AND provider_account_id = ?`, providerItemID, providerAccountID).Scan(&id)
	return id, err
}

func localTransactionIDForProviderTransaction(ctx context.Context, exec sqlExecer, providerItemID string, providerTransactionID string) (string, error) {
	var id string
	err := exec.QueryRowContext(ctx, `
SELECT id
FROM transactions
WHERE provider_item_id = ? AND provider_transaction_id = ?`, providerItemID, providerTransactionID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	return core.NewLocalID("tx_")
}

func (s *SQLiteStore) localRecurringIDForProviderRecurring(ctx context.Context, providerItemID string, providerRecurringID string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
SELECT id
FROM recurring
WHERE provider_item_id = ? AND provider_recurring_id = ?`, providerItemID, providerRecurringID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	return core.NewLocalID("rec_")
}

func (s *SQLiteStore) UpsertSecurity(ctx context.Context, security core.InvestmentSecurity) error {
	now := time.Now().UTC().Format(time.RFC3339)
	id, err := s.localSecurityIDForSecurityID(ctx, security.SecurityID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO securities (
  id, security_id, isin, cusip, sedol, name, ticker_symbol, type,
  close_price, close_price_as_of, currency, created_at, updated_at
) VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, ?)
ON CONFLICT(security_id) DO UPDATE SET
  isin = excluded.isin,
  cusip = excluded.cusip,
  sedol = excluded.sedol,
  name = excluded.name,
  ticker_symbol = excluded.ticker_symbol,
  type = excluded.type,
  close_price = excluded.close_price,
  close_price_as_of = excluded.close_price_as_of,
  currency = excluded.currency,
  updated_at = excluded.updated_at`,
		id, security.SecurityID, ptrNullableString(security.ISIN), ptrNullableString(security.CUSIP), ptrNullableString(security.SEDOL),
		security.Name, ptrNullableString(security.TickerSymbol), ptrNullableString(&security.Type), security.ClosePrice,
		ptrNullableString(security.ClosePriceAsOf), security.Currency, now, now)
	return err
}

func (s *SQLiteStore) localSecurityIDForSecurityID(ctx context.Context, securityID string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM securities WHERE security_id = ?`, securityID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	return core.NewLocalID("sec_")
}

func (s *SQLiteStore) UpsertHolding(ctx context.Context, providerItemID string, holding core.InvestmentHolding) error {
	now := time.Now().UTC().Format(time.RFC3339)
	id, err := s.localHoldingID(ctx, providerItemID, holding.AccountID, holding.SecurityID)
	if err != nil {
		return err
	}
	accountID, err := s.existingAccountIDForProviderAccount(ctx, providerItemID, holding.AccountID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO holdings (
  id, provider_item_id, account_id, provider_account_id, security_id,
  quantity, institution_price, institution_value, cost_basis, currency, created_at, updated_at
) VALUES (?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider_item_id, provider_account_id, security_id) DO UPDATE SET
  account_id = excluded.account_id,
  security_id = excluded.security_id,
  quantity = excluded.quantity,
  institution_price = excluded.institution_price,
  institution_value = excluded.institution_value,
  cost_basis = excluded.cost_basis,
  currency = excluded.currency,
  updated_at = excluded.updated_at`,
		id, providerItemID, accountID, holding.AccountID, holding.SecurityID,
		holding.Quantity, holding.InstitutionPrice, holding.InstitutionValue, holding.CostBasis, holding.Currency, now, now)
	return err
}

func (s *SQLiteStore) localHoldingID(ctx context.Context, providerItemID string, providerAccountID string, securityID string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
SELECT id FROM holdings WHERE provider_item_id = ? AND provider_account_id = ? AND security_id = ?`,
		providerItemID, providerAccountID, securityID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	return core.NewLocalID("hld_")
}

func (s *SQLiteStore) ClearHoldings(ctx context.Context, providerItemID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM holdings WHERE provider_item_id = ?`, providerItemID)
	return err
}

func (s *SQLiteStore) UpsertLiability(ctx context.Context, providerItemID string, liability core.Liability) error {
	now := time.Now().UTC().Format(time.RFC3339)
	id, err := s.localLiabilityID(ctx, providerItemID, liability.AccountID, liability.Type, liability.Name)
	if err != nil {
		return err
	}
	accountID, err := s.existingAccountIDForProviderAccount(ctx, providerItemID, liability.AccountID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO liabilities (
  id, provider_item_id, account_id, provider_account_id, type,
  current_balance, original_balance, currency, name,
  last_payment_date, last_payment_amount, next_payment_due_date, apr, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, ?)
ON CONFLICT(provider_item_id, provider_account_id, type, name) DO UPDATE SET
  account_id = excluded.account_id,
  current_balance = excluded.current_balance,
  original_balance = excluded.original_balance,
  currency = excluded.currency,
  name = excluded.name,
  last_payment_date = excluded.last_payment_date,
  last_payment_amount = excluded.last_payment_amount,
  next_payment_due_date = excluded.next_payment_due_date,
  apr = excluded.apr,
  updated_at = excluded.updated_at`,
		id, providerItemID, accountID, liability.AccountID, liability.Type,
		liability.CurrentBalance, liability.OriginalBalance, liability.Currency, liability.Name,
		ptrNullableString(liability.LastPaymentDate), liability.LastPaymentAmount,
		ptrNullableString(liability.NextPaymentDueDate), liability.APR, now, now)
	return err
}

func (s *SQLiteStore) localLiabilityID(ctx context.Context, providerItemID string, providerAccountID string, liabilityType string, name string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
SELECT id FROM liabilities WHERE provider_item_id = ? AND provider_account_id = ? AND type = ? AND name = ?`,
		providerItemID, providerAccountID, liabilityType, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	return core.NewLocalID("lia_")
}

func (s *SQLiteStore) ClearLiabilities(ctx context.Context, providerItemID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM liabilities WHERE provider_item_id = ?`, providerItemID)
	return err
}

func ptrNullableString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}
