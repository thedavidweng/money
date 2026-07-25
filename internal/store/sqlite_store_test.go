package store

import (
	"context"
	"sort"
	"testing"

	"github.com/thedavidweng/money/internal/core"
)

func TestDemoStoreSeedsDeterministicFeatureCoverage(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	accounts, err := db.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accounts) < 3 {
		t.Fatalf("accounts length = %d, want provider/manual/import examples", len(accounts))
	}

	transactions, err := db.ListTransactions(ctx, &TransactionListQuery{RemovedMode: RemovedExclude, Limit: 50})
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

	removed, err := db.ListTransactions(ctx, &TransactionListQuery{RemovedMode: RemovedOnly, Limit: 50})
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

func TestSQLiteStoreListTransactionsHydratesTags(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	transactions, err := db.ListTransactions(ctx, &TransactionListQuery{RemovedMode: RemovedExclude, Limit: 50})
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}

	// Build a map by ID for easy lookup.
	byID := make(map[string]core.Transaction, len(transactions))
	for _, tx := range transactions {
		byID[tx.ID] = tx
	}

	// Coffee transaction should have exactly 2 tags: "review" and "travel".
	coffee, ok := byID["tx_demo_coffee"]
	if !ok {
		t.Fatal("tx_demo_coffee not found in results")
	}
	if len(coffee.TagIDs) != 2 {
		t.Fatalf("coffee TagIDs length = %d, want 2; TagIDs = %v", len(coffee.TagIDs), coffee.TagIDs)
	}
	if len(coffee.Tags) != 2 {
		t.Fatalf("coffee Tags length = %d, want 2; Tags = %v", len(coffee.Tags), coffee.Tags)
	}

	// Verify the tag names are "review" and "travel".
	tagNames := make([]string, len(coffee.Tags))
	for i, tag := range coffee.Tags {
		tagNames[i] = tag.Name
	}
	sort.Strings(tagNames)
	wantTags := []string{"review", "travel"}
	for i, want := range wantTags {
		if tagNames[i] != want {
			t.Fatalf("coffee tag[%d] name = %q, want %q; all tag names = %v", i, tagNames[i], want, tagNames)
		}
	}

	// Verify TagIDs correspond to the expected tag IDs.
	sort.Strings(coffee.TagIDs)
	wantIDs := []string{"tag_demo_review", "tag_demo_travel"}
	for i, want := range wantIDs {
		if coffee.TagIDs[i] != want {
			t.Fatalf("coffee TagIDs[%d] = %q, want %q", i, coffee.TagIDs[i], want)
		}
	}

	// All other (non-removed) transactions should have empty Tags and TagIDs.
	for _, tx := range transactions {
		if tx.ID == "tx_demo_coffee" {
			continue
		}
		if len(tx.TagIDs) != 0 {
			t.Errorf("transaction %s TagIDs = %v, want empty", tx.ID, tx.TagIDs)
		}
		if len(tx.Tags) != 0 {
			t.Errorf("transaction %s Tags = %v, want empty", tx.ID, tx.Tags)
		}
	}
}
