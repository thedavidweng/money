package store

import (
	"context"
	"testing"
)

func TestDemoStoreSeedsDeterministicFeatureCoverage(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	accounts, err := db.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accounts) < 3 {
		t.Fatalf("accounts length = %d, want provider/manual/import examples", len(accounts))
	}

	transactions, err := db.ListTransactions(ctx, TransactionListQuery{RemovedMode: RemovedExclude, Limit: 50})
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}

	var sawPending, sawReview, sawTags, sawNote, sawRecurring bool
	for _, tx := range transactions {
		if tx.Pending {
			sawPending = true
		}
		if tx.NeedsReview {
			sawReview = true
		}
		if len(tx.TagIDs) > 0 && len(tx.TagIDs) == len(tx.Tags) {
			sawTags = true
		}
		if tx.Note != nil && *tx.Note != "" {
			sawNote = true
		}
		if tx.RecurringTransactionID != nil {
			sawRecurring = true
		}
	}
	if !sawPending || !sawReview || !sawTags || !sawNote || !sawRecurring {
		t.Fatalf("demo coverage pending=%v review=%v tags=%v note=%v recurring=%v", sawPending, sawReview, sawTags, sawNote, sawRecurring)
	}

	removed, err := db.ListTransactions(ctx, TransactionListQuery{RemovedMode: RemovedOnly, Limit: 50})
	if err != nil {
		t.Fatalf("list removed transactions: %v", err)
	}
	if len(removed) == 0 {
		t.Fatal("removed-only list is empty, want soft-deleted fixture")
	}
	if removed[0].TagIDs == nil || removed[0].Tags == nil {
		t.Fatalf("removed transaction tags = %#v/%#v, want empty arrays", removed[0].TagIDs, removed[0].Tags)
	}
}
