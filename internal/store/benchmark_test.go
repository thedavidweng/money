package store

import (
	"context"
	"testing"

	"github.com/thedavidweng/money/internal/core"
)

func BenchmarkListTransactions(b *testing.B) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		b.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	query := TransactionListQuery{RemovedMode: RemovedExclude, Limit: 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.ListTransactions(ctx, &query); err != nil {
			b.Fatalf("list transactions: %v", err)
		}
	}
}

func BenchmarkSearchTransactions(b *testing.B) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		b.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.SearchTransactions(ctx, "Coffee", 50); err != nil {
			b.Fatalf("search transactions: %v", err)
		}
	}
}

func BenchmarkApplyRules(b *testing.B) {
	ctx := context.Background()

	// Pre-flight: verify demo has categories so we can seed rules.
	probe, err := OpenDemo(ctx)
	if err != nil {
		b.Fatalf("open demo: %v", err)
	}
	categories, err := probe.ListCategories(ctx)
	if err != nil || len(categories) == 0 {
		_ = probe.Close()
		b.Skip("demo has no categories")
	}
	catID := categories[0].ID
	_ = probe.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Re-open demo DB each iteration so ApplyRules sees un-categorized
		// transactions every time, measuring fresh-rule application rather
		// than steady-state idempotent updates.
		b.StopTimer()
		db, err := OpenDemo(ctx)
		if err != nil {
			b.Fatalf("open demo: %v", err)
		}
		benchRules := []core.Rule{
			{Name: "bench-coffee", ConditionField: "merchant_name", ConditionOp: "contains", ConditionValue: "Blue Bottle", ActionType: "set_category", ActionValue: catID, Priority: 10, Enabled: true},
			{Name: "bench-rent", ConditionField: "merchant_name", ConditionOp: "contains", ConditionValue: "Rent", ActionType: "set_category", ActionValue: catID, Priority: 9, Enabled: true},
			{Name: "bench-grocery", ConditionField: "merchant_name", ConditionOp: "contains", ConditionValue: "Grocery", ActionType: "set_category", ActionValue: catID, Priority: 8, Enabled: true},
		}
		for i := range benchRules {
			if _, err := db.CreateRule(ctx, &benchRules[i]); err != nil {
				_ = db.Close()
				b.Fatalf("create rule: %v", err)
			}
		}
		b.StartTimer()

		if _, err := db.ApplyRules(ctx); err != nil {
			_ = db.Close()
			b.Fatalf("apply rules: %v", err)
		}

		b.StopTimer()
		_ = db.Close()
		b.StartTimer()
	}
}
