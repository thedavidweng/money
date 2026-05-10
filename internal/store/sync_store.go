package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/thedavidweng/money/internal/core"
	"github.com/thedavidweng/money/internal/providers"
)

func (s *SQLiteStore) UpsertAccount(ctx context.Context, account providers.FinancialAccount) error {
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

func (s *SQLiteStore) UpsertTransaction(ctx context.Context, transaction providers.Transaction) error {
	now := time.Now().UTC().Format(time.RFC3339)
	id, err := s.localTransactionIDForProviderTransaction(ctx, transaction.ProviderItemID, transaction.ProviderTransactionID)
	if err != nil {
		return err
	}
	accountID, err := s.existingAccountIDForProviderAccount(ctx, transaction.ProviderItemID, transaction.ProviderAccountID)
	if err != nil {
		return err
	}
	categorySource := "none"
	var categoryName sql.NullString
	if transaction.ProviderCategory != nil && *transaction.ProviderCategory != "" {
		categorySource = "provider"
		categoryName = sql.NullString{String: *transaction.ProviderCategory, Valid: true}
	}
	_, err = s.db.ExecContext(ctx, `
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

func (s *SQLiteStore) UpsertRecurring(ctx context.Context, recurring providers.Recurring) error {
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

func (s *SQLiteStore) RecordSyncRun(ctx context.Context, run providers.SyncRun) error {
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
	id, err := s.queryAccountIDForProviderAccount(ctx, providerItemID, providerAccountID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("provider account %q for provider item %q must be synced before dependent records", providerAccountID, providerItemID)
	}
	return id, err
}

func (s *SQLiteStore) queryAccountIDForProviderAccount(ctx context.Context, providerItemID string, providerAccountID string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
SELECT id
FROM accounts
WHERE provider_item_id = ? AND provider_account_id = ?`, providerItemID, providerAccountID).Scan(&id)
	return id, err
}

func (s *SQLiteStore) localTransactionIDForProviderTransaction(ctx context.Context, providerItemID string, providerTransactionID string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
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

func ptrNullableString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}
