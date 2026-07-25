package store

import (
	"context"
	"testing"
)

func TestListTransactionsRemovedModes(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// RemovedOnly: only the removed transaction.
	removed, err := db.ListTransactions(ctx, &TransactionListQuery{RemovedMode: RemovedOnly, Limit: 100})
	if err != nil {
		t.Fatalf("removed only: %v", err)
	}
	if len(removed) != 1 || removed[0].ID != "tx_demo_removed" {
		t.Fatalf("removed = %v, want [tx_demo_removed]", removed)
	}

	// RemovedInclude: all 5 transactions.
	all, err := db.ListTransactions(ctx, &TransactionListQuery{RemovedMode: RemovedInclude, Limit: 100})
	if err != nil {
		t.Fatalf("removed include: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("all = %d, want 5", len(all))
	}
}

func TestListTransactionsFilterByAccountAndCategory(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Filter by checking account — has coffee, rent (removed excluded by default).
	checking, err := db.ListTransactions(ctx, &TransactionListQuery{AccountID: "acc_demo_checking", Limit: 100})
	if err != nil {
		t.Fatalf("checking: %v", err)
	}
	if len(checking) != 2 {
		t.Fatalf("checking count = %d, want 2", len(checking))
	}
	for _, tx := range checking {
		if tx.AccountID != "acc_demo_checking" {
			t.Fatalf("tx %s account = %s, want acc_demo_checking", tx.ID, tx.AccountID)
		}
	}

	// Filter by cash account — only the manual adjustment.
	cash, err := db.ListTransactions(ctx, &TransactionListQuery{AccountID: "acc_demo_cash", Limit: 100})
	if err != nil {
		t.Fatalf("cash: %v", err)
	}
	if len(cash) != 1 || cash[0].ID != "tx_demo_cash_adjustment" {
		t.Fatalf("cash = %v, want [tx_demo_cash_adjustment]", cash)
	}

	// Filter by category — cat_demo_food is used by tx_demo_import_grocery.
	food, err := db.ListTransactions(ctx, &TransactionListQuery{CategoryID: "cat_demo_food", Limit: 100})
	if err != nil {
		t.Fatalf("food: %v", err)
	}
	if len(food) != 1 || food[0].ID != "tx_demo_import_grocery" {
		t.Fatalf("food = %v, want [tx_demo_import_grocery]", food)
	}
}

func TestListTransactionsFilterByPendingAndNeedsReview(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	pendingTrue := true
	pending, err := db.ListTransactions(ctx, &TransactionListQuery{Pending: &pendingTrue, Limit: 100})
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != "tx_demo_coffee" {
		t.Fatalf("pending = %v, want [tx_demo_coffee]", pending)
	}

	reviewTrue := true
	review, err := db.ListTransactions(ctx, &TransactionListQuery{NeedsReview: &reviewTrue, Limit: 100})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if len(review) != 1 || review[0].ID != "tx_demo_coffee" {
		t.Fatalf("review = %v, want [tx_demo_coffee]", review)
	}

	pendingFalse := false
	notPending, err := db.ListTransactions(ctx, &TransactionListQuery{Pending: &pendingFalse, Limit: 100})
	if err != nil {
		t.Fatalf("not pending: %v", err)
	}
	if len(notPending) != 3 {
		t.Fatalf("not pending count = %d, want 3", len(notPending))
	}
}

func TestListTransactionsFilterByTagAndDateRange(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Tag filter — tag_demo_travel is attached to tx_demo_coffee only.
	tagged, err := db.ListTransactions(ctx, &TransactionListQuery{TagID: "tag_demo_travel", Limit: 100})
	if err != nil {
		t.Fatalf("tagged: %v", err)
	}
	if len(tagged) != 1 || tagged[0].ID != "tx_demo_coffee" {
		t.Fatalf("tagged = %v, want [tx_demo_coffee]", tagged)
	}
	// Verify tags are hydrated.
	if len(tagged[0].Tags) != 2 {
		t.Fatalf("coffee tags = %d, want 2", len(tagged[0].Tags))
	}

	// Date range — May only (3 non-removed: coffee 05-10, rent 05-01, cash_adjustment 04-15 excluded).
	may, err := db.ListTransactions(ctx, &TransactionListQuery{DateFrom: "2026-05-01", DateTo: "2026-05-31", Limit: 100})
	if err != nil {
		t.Fatalf("may: %v", err)
	}
	if len(may) != 2 {
		t.Fatalf("may count = %d, want 2 (coffee + rent)", len(may))
	}

	// Date from only — everything from April 29 onward.
	fromApril, err := db.ListTransactions(ctx, &TransactionListQuery{DateFrom: "2026-04-29", Limit: 100})
	if err != nil {
		t.Fatalf("from april: %v", err)
	}
	if len(fromApril) != 3 {
		t.Fatalf("from april count = %d, want 3 (grocery + coffee + rent)", len(fromApril))
	}
}

func TestListTransactionsFilterByMerchantAndRecurring(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Merchant LIKE filter — partial match on merchant_name or name.
	coffee, err := db.ListTransactions(ctx, &TransactionListQuery{Merchant: "Blue Bottle", Limit: 100})
	if err != nil {
		t.Fatalf("merchant: %v", err)
	}
	if len(coffee) != 1 || coffee[0].ID != "tx_demo_coffee" {
		t.Fatalf("merchant = %v, want [tx_demo_coffee]", coffee)
	}

	// Case-insensitive partial match on name.
	rent, err := db.ListTransactions(ctx, &TransactionListQuery{Merchant: "rent", Limit: 100})
	if err != nil {
		t.Fatalf("rent: %v", err)
	}
	if len(rent) != 1 || rent[0].ID != "tx_demo_rent" {
		t.Fatalf("rent = %v, want [tx_demo_rent]", rent)
	}

	// Recurring filter — tx_demo_rent has recurring_id set.
	recurringTrue := true
	recurring, err := db.ListTransactions(ctx, &TransactionListQuery{Recurring: &recurringTrue, Limit: 100})
	if err != nil {
		t.Fatalf("recurring: %v", err)
	}
	if len(recurring) != 1 || recurring[0].ID != "tx_demo_rent" {
		t.Fatalf("recurring = %v, want [tx_demo_rent]", recurring)
	}

	recurringFalse := false
	nonRecurring, err := db.ListTransactions(ctx, &TransactionListQuery{Recurring: &recurringFalse, Limit: 100})
	if err != nil {
		t.Fatalf("non-recurring: %v", err)
	}
	if len(nonRecurring) != 3 {
		t.Fatalf("non-recurring count = %d, want 3", len(nonRecurring))
	}
}

func TestListTransactionsPagination(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Get all 4 non-removed transactions.
	all, err := db.ListTransactions(ctx, &TransactionListQuery{Limit: 100})
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("all = %d, want 4", len(all))
	}

	// Page 1: limit 2.
	page1, err := db.ListTransactions(ctx, &TransactionListQuery{Limit: 2})
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1 = %d, want 2", len(page1))
	}

	// Page 2: limit 2, offset 2.
	page2, err := db.ListTransactions(ctx, &TransactionListQuery{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("page2 = %d, want 2", len(page2))
	}

	// Pages should not overlap.
	if page1[0].ID == page2[0].ID {
		t.Fatal("page1 and page2 should not overlap")
	}

	// Page 3: beyond data.
	page3, err := db.ListTransactions(ctx, &TransactionListQuery{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("page3: %v", err)
	}
	if len(page3) != 0 {
		t.Fatalf("page3 = %d, want 0", len(page3))
	}
}

func TestListTransactionsCombinedFilters(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Combine: checking account + pending only → only tx_demo_coffee.
	pendingTrue := true
	combined, err := db.ListTransactions(ctx, &TransactionListQuery{
		AccountID: "acc_demo_checking",
		Pending:   &pendingTrue,
		Limit:     100,
	})
	if err != nil {
		t.Fatalf("combined: %v", err)
	}
	if len(combined) != 1 || combined[0].ID != "tx_demo_coffee" {
		t.Fatalf("combined = %v, want [tx_demo_coffee]", combined)
	}

	// Combine: date range + account → checking in May = coffee + rent.
	mayChecking, err := db.ListTransactions(ctx, &TransactionListQuery{
		AccountID: "acc_demo_checking",
		DateFrom:  "2026-05-01",
		DateTo:    "2026-05-31",
		Limit:     100,
	})
	if err != nil {
		t.Fatalf("may checking: %v", err)
	}
	if len(mayChecking) != 2 {
		t.Fatalf("may checking = %d, want 2", len(mayChecking))
	}
}

func TestListTransactionsDefaultLimit(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Limit=0 should default to 50.
	txDefault, err := db.ListTransactions(ctx, &TransactionListQuery{})
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	// We only have 4 non-removed transactions, so all should come back.
	if len(txDefault) != 4 {
		t.Fatalf("default limit = %d, want 4", len(txDefault))
	}
}

func TestListTransactionsHydratesAccountName(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	txs, err := db.ListTransactions(ctx, &TransactionListQuery{AccountID: "acc_demo_checking", Limit: 100})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(txs) == 0 {
		t.Fatal("no transactions")
	}
	// acc_demo_checking has name "Chase Checking", no alias.
	if txs[0].AccountName != "Chase Checking" {
		t.Fatalf("account name = %q, want %q", txs[0].AccountName, "Chase Checking")
	}

	// acc_demo_cash has alias "Wallet" (takes precedence over name).
	cash, err := db.ListTransactions(ctx, &TransactionListQuery{AccountID: "acc_demo_cash", Limit: 100})
	if err != nil {
		t.Fatalf("cash: %v", err)
	}
	if len(cash) != 1 {
		t.Fatal("no cash transactions")
	}
	if cash[0].AccountName != "Wallet" {
		t.Fatalf("cash account name = %q, want %q (alias takes precedence)", cash[0].AccountName, "Wallet")
	}
}

func TestCashflowSummaryGroupsByPeriod(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Cashflow for April–May 2026.
	periods, err := db.CashflowSummary(ctx, "2026-04-01", "2026-05-31", "monthly", "USD")
	if err != nil {
		t.Fatalf("cashflow: %v", err)
	}
	if len(periods) == 0 {
		t.Fatal("expected at least one period")
	}

	// Verify period labels are present.
	for _, p := range periods {
		if p.Period == "" {
			t.Fatal("period label should not be empty")
		}
		if p.Currency != "USD" {
			t.Fatalf("currency = %q, want USD", p.Currency)
		}
	}
}

func TestNetWorthSumsAllAccounts(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	nw, err := db.NetWorth(ctx)
	if err != nil {
		t.Fatalf("net worth: %v", err)
	}

	// Demo accounts: checking 245678 + cash 12345 + import_card -45012 = 213011
	expected := int64(245678 + 12345 + (-45012))
	if nw.TotalMinor != expected {
		t.Fatalf("net worth = %d, want %d", nw.TotalMinor, expected)
	}
	if nw.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", nw.Currency)
	}
}

func TestSearchTransactionsMatchesNameMerchantAndNote(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Search by merchant name.
	coffee, err := db.SearchTransactions(ctx, "Blue Bottle", 10)
	if err != nil {
		t.Fatalf("search merchant: %v", err)
	}
	if len(coffee) != 1 || coffee[0].ID != "tx_demo_coffee" {
		t.Fatalf("merchant search = %v, want [tx_demo_coffee]", coffee)
	}

	// Search by transaction name (different from merchant).
	grocery, err := db.SearchTransactions(ctx, "Neighborhood", 10)
	if err != nil {
		t.Fatalf("search name: %v", err)
	}
	if len(grocery) != 1 || grocery[0].ID != "tx_demo_import_grocery" {
		t.Fatalf("name search = %v, want [tx_demo_import_grocery]", grocery)
	}

	// Search by note — tx_demo_coffee has note "Ask whether this should be categorized as work travel."
	note, err := db.SearchTransactions(ctx, "work travel", 10)
	if err != nil {
		t.Fatalf("search note: %v", err)
	}
	if len(note) != 1 || note[0].ID != "tx_demo_coffee" {
		t.Fatalf("note search = %v, want [tx_demo_coffee]", note)
	}

	// Search excludes removed transactions.
	removed, err := db.SearchTransactions(ctx, "Removed", 10)
	if err != nil {
		t.Fatalf("search removed: %v", err)
	}
	for _, tx := range removed {
		if tx.ID == "tx_demo_removed" {
			t.Fatal("search should not return removed transactions")
		}
	}
}

func TestListRecurringReturnsDemoStreams(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	recurring, err := db.ListRecurring(ctx)
	if err != nil {
		t.Fatalf("list recurring: %v", err)
	}
	if len(recurring) == 0 {
		t.Fatal("expected at least one recurring stream from demo")
	}

	found := false
	for _, r := range recurring {
		if r.ID == "rec_demo_rent" {
			found = true
			if r.MerchantName != "Rent" {
				t.Fatalf("rent merchant = %q", r.MerchantName)
			}
			if r.Frequency != "monthly" {
				t.Fatalf("rent frequency = %q", r.Frequency)
			}
		}
	}
	if !found {
		t.Fatal("demo rent recurring stream not found")
	}
}

func TestDemoSeedIsSelfConsistent(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Every transaction's account_id should resolve to an account.
	accounts, err := db.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	accountIDs := make(map[string]bool)
	for _, a := range accounts {
		accountIDs[a.ID] = true
	}

	txs, err := db.ListTransactions(ctx, &TransactionListQuery{RemovedMode: RemovedInclude, Limit: 100})
	if err != nil {
		t.Fatalf("list all txs: %v", err)
	}
	for _, tx := range txs {
		if !accountIDs[tx.AccountID] {
			t.Fatalf("tx %s references unknown account %s", tx.ID, tx.AccountID)
		}
	}

	// Every transaction with a provider_item_id should resolve to a provider item.
	items, err := db.ListProviderItems(ctx, ProviderItemQuery{})
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	itemIDs := make(map[string]bool)
	for _, item := range items {
		itemIDs[item.ID] = true
	}
	for _, tx := range txs {
		if tx.Source.ProviderItemID != nil && *tx.Source.ProviderItemID != "" {
			if !itemIDs[*tx.Source.ProviderItemID] {
				t.Fatalf("tx %s references unknown provider item %s", tx.ID, *tx.Source.ProviderItemID)
			}
		}
	}

	// Categories referenced by transactions should exist.
	categories, err := db.ListCategories(ctx)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	catIDs := make(map[string]bool)
	for _, c := range categories {
		catIDs[c.ID] = true
	}
	for _, tx := range txs {
		if tx.CategoryID != nil && *tx.CategoryID != "" {
			if !catIDs[*tx.CategoryID] {
				t.Fatalf("tx %s references unknown category %s", tx.ID, *tx.CategoryID)
			}
		}
	}

	// Tags referenced by transactions should exist.
	tags, err := db.ListTags(ctx)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	tagIDs := make(map[string]bool)
	for _, tag := range tags {
		tagIDs[tag.ID] = true
	}
	for _, tx := range txs {
		for _, tag := range tx.Tags {
			if !tagIDs[tag.ID] {
				t.Fatalf("tx %s references unknown tag %s", tx.ID, tag.ID)
			}
		}
	}

	// Demo should have the expected number of accounts.
	if len(accounts) != 3 {
		t.Fatalf("demo accounts = %d, want 3", len(accounts))
	}
}

func TestListTransactionsExcludesRemovedByDefault(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	txs, err := db.ListTransactions(ctx, &TransactionListQuery{Limit: 100})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, tx := range txs {
		if tx.Removed {
			t.Fatalf("removed transaction %s should be excluded by default", tx.ID)
		}
	}
	// Demo has 5 transactions total, 1 removed → should get 4.
	if len(txs) != 4 {
		t.Fatalf("count = %d, want 4 (demo has 5 txs, 1 removed)", len(txs))
	}
}
