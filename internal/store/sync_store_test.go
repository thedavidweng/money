package store

import (
	"context"
	"fmt"
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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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

func TestSQLiteStoreLatestSyncRunsEmpty(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	runs, err := db.LatestSyncRuns(ctx)
	if err != nil {
		t.Fatalf("latest sync runs: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected 0 sync runs from demo, got %d", len(runs))
	}
}

func TestSQLiteStoreLatestSyncRunsReturnsMostRecentPerProviderItem(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Record two runs for the same provider item — latest should win.
	if err := db.RecordSyncRun(ctx, core.SyncRun{
		Provider:       "plaid",
		ProviderItemID: "pi_demo_plaid",
		StartedAt:      "2026-05-10T10:00:00Z",
		FinishedAt:     "2026-05-10T10:00:02Z",
		Status:         "ok",
	}); err != nil {
		t.Fatalf("record first sync run: %v", err)
	}
	if err := db.RecordSyncRun(ctx, core.SyncRun{
		Provider:       "plaid",
		ProviderItemID: "pi_demo_plaid",
		StartedAt:      "2026-05-11T10:00:00Z",
		FinishedAt:     "2026-05-11T10:00:05Z",
		Status:         "error",
		ErrorCode:      "ITEM_LOGIN_REQUIRED",
		ErrorMessage:   "credentials need refresh",
	}); err != nil {
		t.Fatalf("record second sync run: %v", err)
	}

	runs, err := db.LatestSyncRuns(ctx)
	if err != nil {
		t.Fatalf("latest sync runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	run := runs[0]
	if run.ProviderItemID != "pi_demo_plaid" {
		t.Fatalf("provider item id = %q", run.ProviderItemID)
	}
	if run.Status != "error" {
		t.Fatalf("expected latest run status=error, got %q", run.Status)
	}
	if run.StartedAt != "2026-05-11T10:00:00Z" {
		t.Fatalf("expected latest started_at, got %q", run.StartedAt)
	}
	if run.ErrorCode != "ITEM_LOGIN_REQUIRED" {
		t.Fatalf("error code = %q", run.ErrorCode)
	}
}

func TestSQLiteStoreMarkStuckSyncRunsInterrupted(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Record a stuck run (no finished_at).
	if _, err := db.db.ExecContext(ctx, `
INSERT INTO sync_runs (id, provider, provider_item_id, started_at, status)
VALUES ('sync_stuck', 'plaid', 'pi_demo_plaid', '2026-05-10T10:00:00Z', 'ok')`); err != nil {
		t.Fatalf("insert stuck run: %v", err)
	}

	// Record a completed run.
	if err := db.RecordSyncRun(ctx, core.SyncRun{
		Provider:       "plaid",
		ProviderItemID: "pi_demo_plaid",
		StartedAt:      "2026-05-11T10:00:00Z",
		FinishedAt:     "2026-05-11T10:00:02Z",
		Status:         "ok",
	}); err != nil {
		t.Fatalf("record completed run: %v", err)
	}

	count, err := db.MarkStuckSyncRunsInterrupted(ctx)
	if err != nil {
		t.Fatalf("mark stuck: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 stuck run marked, got %d", count)
	}

	// Verify the stuck run was updated.
	var status, finishedAt string
	if err := db.db.QueryRowContext(ctx, `SELECT status, finished_at FROM sync_runs WHERE id = 'sync_stuck'`).Scan(&status, &finishedAt); err != nil {
		t.Fatalf("query stuck run: %v", err)
	}
	if status != "interrupted" {
		t.Fatalf("status = %q, want interrupted", status)
	}
	if finishedAt == "" {
		t.Fatal("expected finished_at to be set")
	}
}

func TestSQLiteStoreBatchUpsertTransactions(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Set up a linked provider item and account to own our transactions.
	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_batch", Name: "Batch Bank", Provider: "plaid", ProviderInstitutionID: "ins_batch"},
		Item: LinkedItem{
			ID:                     "pi_batch",
			Provider:               "plaid",
			InstitutionID:          "inst_batch",
			ProviderExternalItemID: "item_batch",
			EncryptedAccessToken:   []byte("token"),
			Status:                 "active",
			Products:               []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}

	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID:           "pi_batch",
		ProviderAccountID:        "acc_batch",
		Name:                     "Batch Checking",
		Type:                     "depository",
		Subtype:                  "checking",
		CurrentBalanceMinorUnits: 50000,
		Currency:                 "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}

	// Build 10 transactions for the initial batch insert.
	category := "Shopping"
	makeTransactions := func(names []string, amounts []int64) []core.ProviderTransaction {
		txns := make([]core.ProviderTransaction, len(names))
		for i, name := range names {
			txns[i] = core.ProviderTransaction{
				ProviderItemID:        "pi_batch",
				ProviderTransactionID: fmt.Sprintf("tx_batch_%d", i),
				ProviderAccountID:     "acc_batch",
				Date:                  "2026-06-01",
				AmountMinorUnits:      amounts[i],
				Name:                  name,
				MerchantName:          name,
				ProviderCategory:      &category,
				Currency:              "USD",
			}
		}
		return txns
	}

	names := make([]string, 10)
	amounts := make([]int64, 10)
	for i := 0; i < 10; i++ {
		names[i] = fmt.Sprintf("BatchTxn %d", i)
		amounts[i] = int64(-(i + 1) * 100)
	}

	batch := makeTransactions(names, amounts)
	if err := db.UpsertTransactions(ctx, batch); err != nil {
		t.Fatalf("batch upsert 10 transactions: %v", err)
	}

	// Verify all 10 were inserted.
	results, err := db.SearchTransactions(ctx, "BatchTxn", 50)
	if err != nil {
		t.Fatalf("search transactions: %v", err)
	}
	if len(results) != 10 {
		t.Fatalf("after initial batch upsert: got %d transactions, want 10", len(results))
	}

	// Build a map from name to result for easy lookup.
	resultByName := make(map[string]core.Transaction, len(results))
	for _, r := range results {
		resultByName[r.Name] = r
	}
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("BatchTxn %d", i)
		r, ok := resultByName[name]
		if !ok {
			t.Fatalf("missing transaction %q after initial batch upsert", name)
		}
		if r.AmountMinorUnits != amounts[i] {
			t.Fatalf("transaction %q amount = %d, want %d", name, r.AmountMinorUnits, amounts[i])
		}
	}

	// Now call UpsertTransactions again: modify txns 0-4, leave 5-9 unchanged.
	modNames := make([]string, 10)
	modAmounts := make([]int64, 10)
	for i := 0; i < 10; i++ {
		if i < 5 {
			modNames[i] = fmt.Sprintf("BatchTxn UPDATED %d", i)
			modAmounts[i] = int64(-(i + 1) * 200)
		} else {
			modNames[i] = names[i]
			modAmounts[i] = amounts[i]
		}
	}

	secondBatch := makeTransactions(modNames, modAmounts)
	if err := db.UpsertTransactions(ctx, secondBatch); err != nil {
		t.Fatalf("batch upsert second call: %v", err)
	}

	// Verify all 10 are still present (5 updated + 5 unchanged).
	results2, err := db.SearchTransactions(ctx, "BatchTxn", 50)
	if err != nil {
		t.Fatalf("search transactions after second upsert: %v", err)
	}
	if len(results2) != 10 {
		t.Fatalf("after second batch upsert: got %d transactions, want 10", len(results2))
	}

	resultByName2 := make(map[string]core.Transaction, len(results2))
	for _, r := range results2 {
		resultByName2[r.Name] = r
	}

	// Check the 5 updated transactions (0-4).
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("BatchTxn UPDATED %d", i)
		r, ok := resultByName2[name]
		if !ok {
			t.Fatalf("missing updated transaction %q after second batch upsert", name)
		}
		if r.AmountMinorUnits != modAmounts[i] {
			t.Fatalf("updated transaction %q amount = %d, want %d", name, r.AmountMinorUnits, modAmounts[i])
		}
	}

	// Check the 5 unchanged transactions (5-9).
	for i := 5; i < 10; i++ {
		name := fmt.Sprintf("BatchTxn %d", i)
		r, ok := resultByName2[name]
		if !ok {
			t.Fatalf("missing unchanged transaction %q after second batch upsert", name)
		}
		if r.AmountMinorUnits != amounts[i] {
			t.Fatalf("unchanged transaction %q amount = %d, want %d", name, r.AmountMinorUnits, amounts[i])
		}
	}

	// Ensure the old names for txns 0-4 are gone (they were updated to "BatchTxn UPDATED N").
	for i := 0; i < 5; i++ {
		oldName := fmt.Sprintf("BatchTxn %d", i)
		if _, ok := resultByName2[oldName]; ok {
			t.Fatalf("old transaction %q should have been updated, but still present", oldName)
		}
	}
}
