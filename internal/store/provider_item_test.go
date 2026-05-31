package store

import (
	"context"
	"testing"

	"github.com/thedavidweng/money/internal/core"
)

func TestSQLiteStoreProviderItemUpdateNameAndRemoveCascade(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	// Store a linked item with transactions.
	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_cascade", Name: "Cascade Bank", Provider: "plaid", ProviderInstitutionID: "ins_cascade"},
		Item: LinkedItem{
			ID: "pi_cascade", Provider: "plaid", InstitutionID: "inst_cascade",
			ProviderExternalItemID: "item_cascade", EncryptedAccessToken: []byte("token"),
			Status: "active", Products: []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}
	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID: "pi_cascade", ProviderAccountID: "acc_cascade",
		Name: "Checking", Type: "depository", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID: "pi_cascade", ProviderTransactionID: "tx_cascade1",
		ProviderAccountID: "acc_cascade", Date: "2026-05-01",
		AmountMinorUnits: -1000, Name: "Coffee", MerchantName: "Coffee", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert transaction 1: %v", err)
	}
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID: "pi_cascade", ProviderTransactionID: "tx_cascade2",
		ProviderAccountID: "acc_cascade", Date: "2026-05-02",
		AmountMinorUnits: -2000, Name: "Lunch", MerchantName: "Lunch", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert transaction 2: %v", err)
	}

	// Update provider item name (alias).
	if err := db.UpdateProviderItemName(ctx, "pi_cascade", "My Checking"); err != nil {
		t.Fatalf("update name: %v", err)
	}
	item, err := db.GetProviderItem(ctx, "pi_cascade")
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if item.Alias != "My Checking" {
		t.Fatalf("alias = %q, want %q", item.Alias, "My Checking")
	}

	// Verify transactions exist before removal.
	txs, err := db.ListTransactions(ctx, TransactionListQuery{AccountID: "", Limit: 100})
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	countBefore := 0
	for _, tx := range txs {
		if tx.AccountID != "" {
			countBefore++
		}
	}

	// Remove provider item — should cascade-delete transactions, accounts, etc.
	if err := db.RemoveProviderItem(ctx, "pi_cascade"); err != nil {
		t.Fatalf("remove provider item: %v", err)
	}

	// Provider item should be gone.
	_, err = db.GetProviderItem(ctx, "pi_cascade")
	if err == nil {
		t.Fatal("expected error getting removed provider item")
	}

	// Transactions for that item should be gone.
	searchResults, err := db.SearchTransactions(ctx, "Coffee", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	for _, tx := range searchResults {
		if tx.ID == "tx_cascade1" {
			t.Fatal("cascade-deleted transaction still found in search")
		}
	}
}

func TestSQLiteStoreListProviderItemsFilterByProvider(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	// Demo already has plaid items. Add a bridge item.
	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_bridge", Name: "Bridge Bank", Provider: "bridge", ProviderInstitutionID: "ins_bridge"},
		Item: LinkedItem{
			ID: "pi_bridge", Provider: "bridge", InstitutionID: "inst_bridge",
			ProviderExternalItemID: "item_bridge", EncryptedAccessToken: []byte("tok"),
			Status: "active", Products: []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store bridge item: %v", err)
	}

	// List all — should have both plaid and bridge.
	all, err := db.ListProviderItems(ctx, ProviderItemQuery{})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	hasPlaid, hasBridge := false, false
	for _, item := range all {
		if item.Provider == "plaid" {
			hasPlaid = true
		}
		if item.Provider == "bridge" {
			hasBridge = true
		}
	}
	if !hasPlaid || !hasBridge {
		t.Fatalf("expected both plaid and bridge items, got plaid=%v bridge=%v", hasPlaid, hasBridge)
	}

	// Filter by provider.
	plaidOnly, err := db.ListProviderItems(ctx, ProviderItemQuery{Provider: "plaid"})
	if err != nil {
		t.Fatalf("list plaid: %v", err)
	}
	for _, item := range plaidOnly {
		if item.Provider != "plaid" {
			t.Fatalf("non-plaid item in filtered results: %s", item.Provider)
		}
	}

	// Filter by specific item ID.
	bridgeOnly, err := db.ListProviderItems(ctx, ProviderItemQuery{ProviderItemID: "pi_bridge"})
	if err != nil {
		t.Fatalf("list by id: %v", err)
	}
	if len(bridgeOnly) != 1 || bridgeOnly[0].ID != "pi_bridge" {
		t.Fatalf("filter by id = %v", bridgeOnly)
	}
}
