package store

import (
	"context"
	"strings"
	"testing"

	"github.com/thedavidweng/money/internal/core"
)

func TestSQLiteStoreSyncSinkUpsertsAccountsTransactionsAndRemovedState(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_sync", Name: "Sync Bank", Provider: "plaid", ProviderInstitutionID: "ins_sync"},
		Item: LinkedItem{
			ID:                     "pi_sync",
			Provider:               "plaid",
			InstitutionID:          "inst_sync",
			ProviderExternalItemID: "item_sync",
			EncryptedAccessToken:   []byte("token"),
			Status:                 "active",
			Products:               []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}

	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID:           "pi_sync",
		ProviderAccountID:        "acc_provider",
		Name:                     "Checking",
		Type:                     "depository",
		Subtype:                  "checking",
		CurrentBalanceMinorUnits: 12345,
		Currency:                 "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}
	var localAccountID string
	if err := db.db.QueryRowContext(ctx, `SELECT id FROM accounts WHERE provider_item_id = ? AND provider_account_id = ?`, "pi_sync", "acc_provider").Scan(&localAccountID); err != nil {
		t.Fatalf("query synced account id: %v", err)
	}
	if !strings.HasPrefix(localAccountID, "acc_") {
		t.Fatalf("synced account id = %q, want local acc_ id", localAccountID)
	}
	category := "Food"
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID:        "pi_sync",
		ProviderTransactionID: "tx_provider",
		ProviderAccountID:     "acc_provider",
		Date:                  "2026-05-10",
		AmountMinorUnits:      -1234,
		Name:                  "Lunch",
		MerchantName:          "Lunch",
		ProviderCategory:      &category,
		Currency:              "USD",
	}); err != nil {
		t.Fatalf("upsert transaction: %v", err)
	}
	var localTransactionID, transactionAccountID string
	if err := db.db.QueryRowContext(ctx, `SELECT id, account_id FROM transactions WHERE provider_item_id = ? AND provider_transaction_id = ?`, "pi_sync", "tx_provider").Scan(&localTransactionID, &transactionAccountID); err != nil {
		t.Fatalf("query synced transaction id: %v", err)
	}
	if !strings.HasPrefix(localTransactionID, "tx_") {
		t.Fatalf("synced transaction id = %q, want local tx_ id", localTransactionID)
	}
	if transactionAccountID != localAccountID {
		t.Fatalf("synced transaction account_id = %q, want %q", transactionAccountID, localAccountID)
	}
	if err := db.MarkTransactionRemoved(ctx, "pi_sync", "tx_provider"); err != nil {
		t.Fatalf("mark removed: %v", err)
	}

	removed, err := db.ListTransactions(ctx, TransactionListQuery{RemovedMode: RemovedOnly, Limit: 10})
	if err != nil {
		t.Fatalf("list removed: %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("removed transactions = %d, want demo removed plus synced removed", len(removed))
	}
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID:        "pi_sync",
		ProviderTransactionID: "tx_provider",
		ProviderAccountID:     "acc_provider",
		Date:                  "2026-05-11",
		AmountMinorUnits:      -2000,
		Name:                  "Lunch updated",
		MerchantName:          "Lunch",
		Currency:              "USD",
	}); err != nil {
		t.Fatalf("upsert modified transaction: %v", err)
	}
	active, err := db.SearchTransactions(ctx, "Lunch updated", 10)
	if err != nil {
		t.Fatalf("search transactions: %v", err)
	}
	if len(active) != 1 || active[0].Amount != "-20.00" {
		t.Fatalf("active transactions = %#v", active)
	}
}

func TestSQLiteStoreRecordsSyncRunCounts(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	if err := db.RecordSyncRun(ctx, core.SyncRun{
		Provider:             "plaid",
		ProviderItemID:       "pi_demo_plaid",
		StartedAt:            "2026-05-10T10:00:00Z",
		FinishedAt:           "2026-05-10T10:00:02Z",
		Status:               "ok",
		AccountsSeen:         2,
		TransactionsAdded:    3,
		TransactionsModified: 4,
		TransactionsRemoved:  5,
		RecurringStreamsSeen: 6,
	}); err != nil {
		t.Fatalf("record sync run: %v", err)
	}

	var provider string
	var accountsSeen, added, modified, removed, recurringSeen int
	if err := db.db.QueryRowContext(ctx, `
SELECT provider, accounts_seen, transactions_added, transactions_modified, transactions_removed, recurring_seen
FROM sync_runs
WHERE provider_item_id = ?
ORDER BY started_at DESC
LIMIT 1`, "pi_demo_plaid").Scan(&provider, &accountsSeen, &added, &modified, &removed, &recurringSeen); err != nil {
		t.Fatalf("query sync run: %v", err)
	}
	if provider != "plaid" || accountsSeen != 2 || added != 3 || modified != 4 || removed != 5 || recurringSeen != 6 {
		t.Fatalf("sync run counts = provider %q accounts %d added %d modified %d removed %d recurring %d", provider, accountsSeen, added, modified, removed, recurringSeen)
	}
}

func TestSQLiteStoreListsProviderItemsForSync(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	items, err := db.ListProviderItems(ctx, ProviderItemQuery{Provider: "plaid"})
	if err != nil {
		t.Fatalf("list provider items: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected demo provider item")
	}
	if items[0].EncryptedAccessToken == nil {
		t.Fatalf("provider item token was not loaded: %#v", items[0])
	}
}
