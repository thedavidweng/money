package store

import (
	"context"
)

func (s *SQLiteStore) SeedDemo(ctx context.Context) error {
	now := "2026-05-10T12:00:00Z"
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO institutions (id, name, provider, provider_institution_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, []any{"inst_demo_chase", "Chase", "plaid", "ins_3", now, now}},
		{`INSERT INTO institutions (id, name, provider, provider_institution_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, []any{"inst_demo_import", "Imported statements", nil, nil, now, now}},
		{`INSERT INTO provider_items (id, provider, institution_id, provider_external_item_id, encrypted_access_token, status, products_json, transaction_cursor, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"pi_demo_plaid", "plaid", "inst_demo_chase", "item_demo_plaid", []byte("demo-token-placeholder"), "active", `["transactions"]`, "cursor_demo", now, now}},

		{`INSERT INTO categories (id, name, group_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, []any{"cat_demo_food", "Restaurants", "Food", now, now}},
		{`INSERT INTO categories (id, name, group_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, []any{"cat_demo_income", "Paychecks", "Income", now, now}},
		{`INSERT INTO tags (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`, []any{"tag_demo_review", "review", now, now}},
		{`INSERT INTO tags (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`, []any{"tag_demo_travel", "travel", now, now}},

		{`INSERT INTO accounts (id, provider_item_id, institution_id, provider_account_id, source_kind, name, official_name, mask, type, subtype, current_balance_minor_units, available_balance_minor_units, currency, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"acc_demo_checking", "pi_demo_plaid", "inst_demo_chase", "plaid_checking_1", "provider", "Chase Checking", "Chase Total Checking", "0000", "depository", "checking", 245678, 245678, "USD", now, now}},
		{`INSERT INTO accounts (id, source_kind, name, alias, type, subtype, current_balance_minor_units, currency, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"acc_demo_cash", "manual", "Cash Wallet", "Wallet", "depository", "cash", 12345, "USD", now, now}},
		{`INSERT INTO accounts (id, institution_id, source_kind, import_source_id, import_batch_id, name, type, subtype, current_balance_minor_units, currency, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"acc_demo_import_card", "inst_demo_import", "import", "import_demo_apple_card", "batch_demo_2026_05", "Apple Card Import", "credit", "credit_card", -45012, "USD", now, now}},

		{`INSERT INTO recurring (id, provider_item_id, provider_recurring_id, account_id, provider_account_id, source_kind, merchant_name, average_amount_minor_units, currency, frequency, next_date, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"rec_demo_rent", "pi_demo_plaid", "stream_demo_rent", "acc_demo_checking", "plaid_checking_1", "provider", "Rent", -180000, "USD", "monthly", "2026-06-01", now, now}},

		{`INSERT INTO transactions (id, account_id, provider_item_id, provider_transaction_id, provider_account_id, source_kind, date, authorized_date, amount_minor_units, currency, name, merchant_name, category_id, category_name, category_source, provider_category, provider_subcategory, pending, removed, needs_review, note, recurring_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"tx_demo_coffee", "acc_demo_checking", "pi_demo_plaid", "plaid_tx_coffee", "plaid_checking_1", "provider", "2026-05-10", "2026-05-10", -625, "USD", "Blue Bottle Coffee", "Blue Bottle Coffee", nil, "Restaurants", "provider", "Food and Drink", "Coffee Shop", 1, 0, 1, "Ask whether this should be categorized as work travel.", nil, now, now}},
		{`INSERT INTO transactions (id, account_id, provider_item_id, provider_transaction_id, provider_account_id, source_kind, date, amount_minor_units, currency, name, merchant_name, category_id, category_name, category_source, pending, removed, needs_review, recurring_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"tx_demo_rent", "acc_demo_checking", "pi_demo_plaid", "plaid_tx_rent", "plaid_checking_1", "provider", "2026-05-01", -180000, "USD", "Rent", "Rent", nil, "Rent", "provider", 0, 0, 0, "rec_demo_rent", now, now}},
		{`INSERT INTO transactions (id, account_id, source_kind, import_source_id, import_batch_id, source_row_hash, date, amount_minor_units, currency, name, merchant_name, category_id, category_name, category_source, pending, removed, needs_review, note, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"tx_demo_import_grocery", "acc_demo_import_card", "import", "import_demo_apple_card", "batch_demo_2026_05", "rowhash_demo_grocery", "2026-04-29", -5421, "USD", "Neighborhood Grocery", "Neighborhood Grocery", "cat_demo_food", "Restaurants", "local", 0, 0, 0, "Imported Apple Card row.", now, now}},
		{`INSERT INTO transactions (id, account_id, source_kind, date, amount_minor_units, currency, name, merchant_name, category_id, category_name, category_source, pending, removed, needs_review, note, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"tx_demo_cash_adjustment", "acc_demo_cash", "manual", "2026-04-15", 2000, "USD", "Cash adjustment", "Cash adjustment", nil, nil, "none", 0, 0, 0, "Manual balance correction.", now, now}},
		{`INSERT INTO transactions (id, account_id, provider_item_id, provider_transaction_id, provider_account_id, source_kind, date, amount_minor_units, currency, name, merchant_name, category_id, category_name, category_source, pending, removed, needs_review, removed_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, []any{"tx_demo_removed", "acc_demo_checking", "pi_demo_plaid", "plaid_tx_removed", "plaid_checking_1", "provider", "2026-04-01", -999, "USD", "Removed test transaction", "Removed merchant", nil, nil, "none", 0, 1, 0, now, now, now}},
		{`INSERT INTO transaction_tags (transaction_id, tag_id, created_at) VALUES (?, ?, ?)`, []any{"tx_demo_coffee", "tag_demo_review", now}},
		{`INSERT INTO transaction_tags (transaction_id, tag_id, created_at) VALUES (?, ?, ?)`, []any{"tx_demo_coffee", "tag_demo_travel", now}},
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
