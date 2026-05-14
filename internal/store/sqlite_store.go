package store

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum"

	"github.com/thedavidweng/money/internal/core"
)

//go:embed migrations/0001_initial.sql
var initialMigration string

//go:embed migrations/0002_investments_liabilities.sql
var investmentsLiabilitiesMigration string

type SQLiteStore struct {
	db *sql.DB
}

func OpenEncrypted(ctx context.Context, path string, key []byte) (*SQLiteStore, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("database encryption key must be 32 bytes")
	}
	if path == "" {
		return nil, fmt.Errorf("database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(path) + "?vfs=adiantum&hexkey=" + hex.EncodeToString(key) + "&_pragma=temp_store(memory)"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &SQLiteStore{db: db}
	if err := store.runMigrations(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func OpenDemo(ctx context.Context) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	store := &SQLiteStore{db: db}
	if err := store.runMigrations(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.SeedDemo(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) runMigrations(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'institutions'`).Scan(&count); err != nil {
		return fmt.Errorf("check migration state: %w", err)
	}
	if count == 0 {
		if _, err := s.db.ExecContext(ctx, initialMigration); err != nil {
			return fmt.Errorf("run initial migration: %w", err)
		}
	}
	if err := s.migrateProviderItemAlias(ctx); err != nil {
		return err
	}
	if err := s.migrateInvestmentsLiabilities(ctx); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) migrateInvestmentsLiabilities(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'securities'`).Scan(&count); err != nil {
		return fmt.Errorf("check investments_liabilities migration state: %w", err)
	}
	if count == 0 {
		if _, err := s.db.ExecContext(ctx, investmentsLiabilitiesMigration); err != nil {
			return fmt.Errorf("run investments_liabilities migration: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStore) migrateProviderItemAlias(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(provider_items)`)
	if err != nil {
		return fmt.Errorf("check provider_items columns: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "alias" {
			return nil
		}
	}
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE provider_items ADD COLUMN alias TEXT`); err != nil {
		return fmt.Errorf("add provider_items alias column: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListAccounts(ctx context.Context) ([]core.Account, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  a.id, a.name, COALESCE(a.official_name, ''), COALESCE(a.alias, ''), COALESCE(a.mask, ''),
  COALESCE(a.institution_id, ''), a.type, COALESCE(a.subtype, ''),
  a.current_balance_minor_units, a.available_balance_minor_units, a.available_credit_minor_units,
  a.currency, a.source_kind, COALESCE(pi.provider, ''), a.provider_item_id,
  COALESCE(pi.provider_external_item_id, ''), a.provider_account_id,
  a.import_source_id, a.import_batch_id, a.hidden, a.updated_at
FROM accounts a
LEFT JOIN provider_items pi ON pi.id = a.provider_item_id
ORDER BY a.hidden ASC, a.type ASC, a.name ASC, a.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []core.Account{}
	for rows.Next() {
		var account core.Account
		var availableBalance, availableCredit sql.NullInt64
		var provider, providerItemID, providerExternalItemID, providerAccountID sql.NullString
		var importSourceID, importBatchID sql.NullString
		if err := rows.Scan(
			&account.ID, &account.Name, &account.OfficialName, &account.Alias, &account.Mask,
			&account.InstitutionID, &account.Type, &account.Subtype,
			&account.CurrentBalanceMinorUnits, &availableBalance, &availableCredit,
			&account.Currency, &account.Source.Kind, &provider, &providerItemID,
			&providerExternalItemID, &providerAccountID,
			&importSourceID, &importBatchID, &account.Hidden, &account.UpdatedAt,
		); err != nil {
			return nil, err
		}
		account.DisplayName = displayName(account)
		account.CurrentBalance = core.FormatMinorUnits(account.CurrentBalanceMinorUnits, account.Currency)
		account.AvailableBalanceMinorUnits, account.AvailableBalance = nullableMoney(availableBalance, account.Currency)
		account.AvailableCreditMinorUnits, account.AvailableCredit = nullableMoney(availableCredit, account.Currency)
		account.Source = source(account.Source.Kind, provider, providerItemID, providerExternalItemID, nullableString(account.InstitutionID), providerAccountID, sql.NullString{}, importSourceID, importBatchID)
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (s *SQLiteStore) CreateManualAccount(ctx context.Context, account core.Account) (core.Account, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if account.ID == "" {
		id, err := core.NewLocalID("acc_")
		if err != nil {
			return core.Account{}, err
		}
		account.ID = id
	}
	if account.Source.Kind == "" {
		account.Source.Kind = "manual"
	}
	if account.UpdatedAt == "" {
		account.UpdatedAt = now
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO accounts (
  id, source_kind, name, official_name, alias, type, subtype,
  current_balance_minor_units, currency, created_at, updated_at
) VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, ?, ?)`,
		account.ID, account.Source.Kind, account.Name, account.OfficialName, account.Alias, account.Type, account.Subtype,
		account.CurrentBalanceMinorUnits, account.Currency, now, account.UpdatedAt)
	if err != nil {
		return core.Account{}, err
	}
	account.DisplayName = displayName(account)
	account.CurrentBalance = core.FormatMinorUnits(account.CurrentBalanceMinorUnits, account.Currency)
	account.Source = source("manual", sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{})
	return account, nil
}

func (s *SQLiteStore) ListTransactions(ctx context.Context, query TransactionListQuery) ([]core.Transaction, error) {
	clauses := []string{"1=1"}
	args := []any{}
	switch query.RemovedMode {
	case "", RemovedExclude:
		clauses = append(clauses, "t.removed = 0")
	case RemovedOnly:
		clauses = append(clauses, "t.removed = 1")
	case RemovedInclude:
	default:
		return nil, fmt.Errorf("unsupported removed mode %q", query.RemovedMode)
	}
	if query.AccountID != "" {
		clauses = append(clauses, "t.account_id = ?")
		args = append(args, query.AccountID)
	}
	if query.CategoryID != "" {
		clauses = append(clauses, "t.category_id = ?")
		args = append(args, query.CategoryID)
	}
	if query.Merchant != "" {
		clauses = append(clauses, "(t.merchant_name LIKE ? OR t.name LIKE ?)")
		needle := "%" + query.Merchant + "%"
		args = append(args, needle, needle)
	}
	if query.TagID != "" {
		clauses = append(clauses, `EXISTS (SELECT 1 FROM transaction_tags tt WHERE tt.transaction_id = t.id AND tt.tag_id = ?)`)
		args = append(args, query.TagID)
	}
	if query.NeedsReview != nil {
		clauses = append(clauses, "t.needs_review = ?")
		args = append(args, boolInt(*query.NeedsReview))
	}
	if query.Pending != nil {
		clauses = append(clauses, "t.pending = ?")
		args = append(args, boolInt(*query.Pending))
	}
	if query.Recurring != nil {
		if *query.Recurring {
			clauses = append(clauses, "t.recurring_id IS NOT NULL")
		} else {
			clauses = append(clauses, "t.recurring_id IS NULL")
		}
	}
	if query.DateFrom != "" {
		clauses = append(clauses, "t.date >= ?")
		args = append(args, query.DateFrom)
	}
	if query.DateTo != "" {
		clauses = append(clauses, "t.date <= ?")
		args = append(args, query.DateTo)
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	args = append(args, limit, query.Offset)
	return s.queryTransactions(ctx, strings.Join(clauses, " AND "), args)
}

func (s *SQLiteStore) SearchTransactions(ctx context.Context, query string, limit int) ([]core.Transaction, error) {
	if limit <= 0 {
		limit = 50
	}
	needle := "%" + query + "%"
	return s.queryTransactions(ctx, "t.removed = 0 AND (t.name LIKE ? OR t.merchant_name LIKE ? OR t.note LIKE ?)", []any{needle, needle, needle, limit, 0})
}

func (s *SQLiteStore) queryTransactions(ctx context.Context, where string, args []any) ([]core.Transaction, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  t.id, t.account_id, t.date, t.authorized_date, t.datetime, t.authorized_datetime,
  t.amount_minor_units, t.currency, t.name, COALESCE(t.merchant_name, ''),
  t.category_id, t.category_name, t.category_source, t.provider_category, t.provider_subcategory,
  t.pending, t.removed, t.needs_review, t.note, t.recurring_id, t.updated_at,
  t.source_kind, COALESCE(pi.provider, ''), t.provider_item_id, COALESCE(pi.provider_external_item_id, ''),
  COALESCE(pi.institution_id, ''), t.provider_account_id, t.provider_transaction_id,
  t.import_source_id, t.import_batch_id,
  COALESCE(NULLIF(a.alias, ''), NULLIF(a.name, ''), NULLIF(a.official_name, ''), '') AS account_name
FROM transactions t
LEFT JOIN provider_items pi ON pi.id = t.provider_item_id
LEFT JOIN accounts a ON a.id = t.account_id
WHERE `+where+`
ORDER BY t.date DESC, t.pending DESC, t.id ASC
LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []core.Transaction{}
	for rows.Next() {
		var tx core.Transaction
		var authorizedDate, datetimeValue, authorizedDatetime sql.NullString
		var categoryID, categoryName, providerCategory, providerSubcategory sql.NullString
		var note, recurringID sql.NullString
		var provider, providerItemID, providerExternalItemID, institutionID, providerAccountID, providerTransactionID sql.NullString
		var importSourceID, importBatchID sql.NullString
		if err := rows.Scan(
			&tx.ID, &tx.AccountID, &tx.Date, &authorizedDate, &datetimeValue, &authorizedDatetime,
			&tx.AmountMinorUnits, &tx.Currency, &tx.Name, &tx.MerchantName,
			&categoryID, &categoryName, &tx.CategorySource, &providerCategory, &providerSubcategory,
			&tx.Pending, &tx.Removed, &tx.NeedsReview, &note, &recurringID, &tx.LastChangedAt,
			&tx.Source.Kind, &provider, &providerItemID, &providerExternalItemID,
			&institutionID, &providerAccountID, &providerTransactionID,
			&importSourceID, &importBatchID, &tx.AccountName,
		); err != nil {
			return nil, err
		}
		tx.AuthorizedDate = stringPtr(authorizedDate)
		tx.Datetime = stringPtr(datetimeValue)
		tx.AuthorizedDatetime = stringPtr(authorizedDatetime)
		tx.Amount = core.FormatMinorUnits(tx.AmountMinorUnits, tx.Currency)
		tx.CategoryID = stringPtr(categoryID)
		tx.CategoryName = stringPtr(categoryName)
		tx.ProviderCategory = stringPtr(providerCategory)
		tx.ProviderSubcategory = stringPtr(providerSubcategory)
		tx.Note = stringPtr(note)
		tx.RecurringTransactionID = stringPtr(recurringID)
		tx.Source = source(tx.Source.Kind, provider, providerItemID, providerExternalItemID, institutionID, providerAccountID, providerTransactionID, importSourceID, importBatchID)
		transactions = append(transactions, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.hydrateTransactionTags(ctx, transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

func (s *SQLiteStore) ListCategories(ctx context.Context) ([]core.Category, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, group_name, hidden, updated_at FROM categories ORDER BY hidden ASC, group_name ASC, name ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	categories := []core.Category{}
	for rows.Next() {
		var category core.Category
		var groupName sql.NullString
		if err := rows.Scan(&category.ID, &category.Name, &groupName, &category.Hidden, &category.UpdatedAt); err != nil {
			return nil, err
		}
		category.GroupName = stringPtr(groupName)
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (s *SQLiteStore) ListTags(ctx context.Context) ([]core.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, updated_at FROM tags ORDER BY name ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []core.Tag{}
	for rows.Next() {
		var tag core.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.UpdatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *SQLiteStore) ListRecurring(ctx context.Context) ([]core.Recurring, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT r.id, r.account_id, r.merchant_name, r.average_amount_minor_units, r.currency, r.frequency, r.next_date,
       r.source_kind, COALESCE(pi.provider, ''), r.provider_item_id, COALESCE(pi.provider_external_item_id, ''),
       COALESCE(pi.institution_id, ''), r.provider_account_id, r.updated_at
FROM recurring r
LEFT JOIN provider_items pi ON pi.id = r.provider_item_id
ORDER BY r.next_date ASC, r.merchant_name ASC, r.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	recurringItems := []core.Recurring{}
	for rows.Next() {
		var item core.Recurring
		var nextDate sql.NullString
		var provider, providerItemID, providerExternalItemID, institutionID, providerAccountID sql.NullString
		var kind string
		if err := rows.Scan(&item.ID, &item.AccountID, &item.MerchantName, &item.AverageAmountMinorUnits, &item.Currency, &item.Frequency, &nextDate, &kind, &provider, &providerItemID, &providerExternalItemID, &institutionID, &providerAccountID, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.AverageAmount = core.FormatMinorUnits(item.AverageAmountMinorUnits, item.Currency)
		item.NextDate = stringPtr(nextDate)
		item.Source = source(kind, provider, providerItemID, providerExternalItemID, institutionID, providerAccountID, sql.NullString{}, sql.NullString{}, sql.NullString{})
		recurringItems = append(recurringItems, item)
	}
	return recurringItems, rows.Err()
}

func (s *SQLiteStore) ListHoldings(ctx context.Context) ([]core.InvestmentHolding, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT h.id, h.account_id, h.security_id, h.quantity, h.institution_price, h.institution_value, h.cost_basis, h.currency, h.updated_at
FROM holdings h
ORDER BY h.institution_value DESC, h.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.InvestmentHolding{}
	for rows.Next() {
		var h core.InvestmentHolding
		var costBasis sql.NullFloat64
		if err := rows.Scan(&h.ID, &h.AccountID, &h.SecurityID, &h.Quantity, &h.InstitutionPrice, &h.InstitutionValue, &costBasis, &h.Currency, &h.UpdatedAt); err != nil {
			return nil, err
		}
		if costBasis.Valid {
			h.CostBasis = &costBasis.Float64
		}
		items = append(items, h)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) ListSecurities(ctx context.Context) ([]core.InvestmentSecurity, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, security_id, isin, cusip, sedol, name, ticker_symbol, type, close_price, close_price_as_of, currency, updated_at
FROM securities
ORDER BY name ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.InvestmentSecurity{}
	for rows.Next() {
		var sec core.InvestmentSecurity
		var isin, cusip, sedol, ticker, typ, closePriceAsOf sql.NullString
		if err := rows.Scan(&sec.ID, &sec.SecurityID, &isin, &cusip, &sedol, &sec.Name, &ticker, &typ, &sec.ClosePrice, &closePriceAsOf, &sec.Currency, &sec.UpdatedAt); err != nil {
			return nil, err
		}
		sec.ISIN = stringPtr(isin)
		sec.CUSIP = stringPtr(cusip)
		sec.SEDOL = stringPtr(sedol)
		if ticker.Valid && ticker.String != "" {
			sec.TickerSymbol = &ticker.String
		}
		if typ.Valid && typ.String != "" {
			sec.Type = typ.String
		}
		sec.ClosePriceAsOf = stringPtr(closePriceAsOf)
		items = append(items, sec)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) ListLiabilities(ctx context.Context) ([]core.Liability, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, account_id, type, current_balance, original_balance, currency, name,
       last_payment_date, last_payment_amount, next_payment_due_date, apr, updated_at
FROM liabilities
ORDER BY current_balance DESC, name ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Liability{}
	for rows.Next() {
		var l core.Liability
		var originalBalance, lastPaymentAmount, apr sql.NullFloat64
		var lastPaymentDate, nextPaymentDueDate sql.NullString
		if err := rows.Scan(&l.ID, &l.AccountID, &l.Type, &l.CurrentBalance, &originalBalance, &l.Currency, &l.Name,
			&lastPaymentDate, &lastPaymentAmount, &nextPaymentDueDate, &apr, &l.UpdatedAt); err != nil {
			return nil, err
		}
		if originalBalance.Valid {
			l.OriginalBalance = &originalBalance.Float64
		}
		if lastPaymentAmount.Valid {
			l.LastPaymentAmount = &lastPaymentAmount.Float64
		}
		if apr.Valid {
			l.APR = &apr.Float64
		}
		l.LastPaymentDate = stringPtr(lastPaymentDate)
		l.NextPaymentDueDate = stringPtr(nextPaymentDueDate)
		items = append(items, l)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) hydrateTransactionTags(ctx context.Context, transactions []core.Transaction) error {
	for i := range transactions {
		tags, err := s.tagsForTransaction(ctx, transactions[i].ID)
		if err != nil {
			return err
		}
		transactions[i].Tags = tags
		transactions[i].TagIDs = []string{}
		for _, tag := range tags {
			transactions[i].TagIDs = append(transactions[i].TagIDs, tag.ID)
		}
	}
	return nil
}

func (s *SQLiteStore) tagsForTransaction(ctx context.Context, transactionID string) ([]core.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT tags.id, tags.name, tags.updated_at
FROM tags
JOIN transaction_tags ON transaction_tags.tag_id = tags.id
WHERE transaction_tags.transaction_id = ?
ORDER BY tags.name ASC, tags.id ASC`, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []core.Tag{}
	for rows.Next() {
		var tag core.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.UpdatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func nullableMoney(value sql.NullInt64, currency string) (*int64, *string) {
	if !value.Valid {
		return nil, nil
	}
	formatted := core.FormatMinorUnits(value.Int64, currency)
	return &value.Int64, &formatted
}

func source(kind string, provider, providerItemID, providerExternalItemID, institutionID, providerAccountID, providerTransactionID, importSourceID, importBatchID sql.NullString) core.Source {
	return core.Source{
		Kind:                   kind,
		Provider:               stringPtr(provider),
		ProviderItemID:         stringPtr(providerItemID),
		ProviderExternalItemID: stringPtr(providerExternalItemID),
		InstitutionID:          stringPtr(institutionID),
		ProviderAccountID:      stringPtr(providerAccountID),
		ProviderTransactionID:  stringPtr(providerTransactionID),
		ImportSourceID:         stringPtr(importSourceID),
		ImportBatchID:          stringPtr(importBatchID),
	}
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func stringPtr(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	return &value.String
}

func displayName(account core.Account) string {
	if account.Alias != "" {
		return account.Alias
	}
	if account.Name != "" {
		return account.Name
	}
	return account.OfficialName
}

func (s *SQLiteStore) UpsertImportedAccount(ctx context.Context, account core.Account) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if account.UpdatedAt == "" {
		account.UpdatedAt = now
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO accounts (
  id, source_kind, import_source_id, import_batch_id, name, official_name, alias,
  type, subtype, current_balance_minor_units, currency, hidden, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, 0, ?, ?)`,
		account.ID, account.Source.Kind,
		nullString(account.Source.ImportSourceID), nullString(account.Source.ImportBatchID),
		account.Name, account.OfficialName, account.Alias,
		account.Type, account.Subtype,
		account.CurrentBalanceMinorUnits, account.Currency,
		now, account.UpdatedAt)
	return err
}

func (s *SQLiteStore) UpsertImportedTransaction(ctx context.Context, tx core.Transaction, sourceRowHash string) (bool, []string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	lastChanged := tx.LastChangedAt
	if lastChanged == "" {
		lastChanged = now
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO transactions (
  id, account_id, source_kind, import_source_id, import_batch_id, source_row_hash,
  date, amount_minor_units, currency, name, merchant_name,
  category_id, category_name, category_source, provider_category, provider_subcategory,
  pending, removed, needs_review, note, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), ?, 0, 0, NULLIF(?, ''), ?, ?)`,
		tx.ID, tx.AccountID, tx.Source.Kind,
		nullString(tx.Source.ImportSourceID), nullString(tx.Source.ImportBatchID), sourceRowHash,
		tx.Date, tx.AmountMinorUnits, tx.Currency, tx.Name, nullString(&tx.MerchantName),
		nullString(tx.CategoryID), nullString(tx.CategoryName), tx.CategorySource,
		nullString(tx.ProviderCategory), nullString(tx.ProviderSubcategory),
		boolInt(tx.Pending), nullString(tx.Note), now, lastChanged)
	if err != nil {
		if isUniqueConstraintError(err) {
			return false, nil, nil // same-batch duplicate, skipped
		}
		return false, nil, err
	}

	possibleDups, err := s.findPossibleDuplicates(ctx, tx)
	return true, possibleDups, err
}

func (s *SQLiteStore) findPossibleDuplicates(ctx context.Context, tx core.Transaction) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id FROM transactions
WHERE account_id = ? AND date = ? AND amount_minor_units = ? AND id != ?
  AND (import_source_id IS DISTINCT FROM ? OR import_batch_id IS DISTINCT FROM ?)
LIMIT 5`,
		tx.AccountID, tx.Date, tx.AmountMinorUnits, tx.ID,
		nullString(tx.Source.ImportSourceID), nullString(tx.Source.ImportBatchID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") || strings.Contains(msg, "constraint violation")
}

func nullString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
