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
		if _, err := db.ListTransactions(ctx, query); err != nil {
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
	db, err := OpenDemo(ctx)
	if err != nil {
		b.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	categories, err := db.ListCategories(ctx)
	if err != nil || len(categories) == 0 {
		b.Skip("demo has no categories")
	}
	catID := categories[0].ID

	rules := []core.Rule{
		{Name: "bench-coffee", ConditionField: "merchant_name", ConditionOp: "contains", ConditionValue: "Blue Bottle", ActionType: "set_category", ActionValue: catID, Priority: 10, Enabled: true},
		{Name: "bench-rent", ConditionField: "merchant_name", ConditionOp: "contains", ConditionValue: "Rent", ActionType: "set_category", ActionValue: catID, Priority: 9, Enabled: true},
		{Name: "bench-grocery", ConditionField: "merchant_name", ConditionOp: "contains", ConditionValue: "Grocery", ActionType: "set_category", ActionValue: catID, Priority: 8, Enabled: true},
	}
	for _, r := range rules {
		if _, err := db.CreateRule(ctx, r); err != nil {
			b.Fatalf("create rule: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.ApplyRules(ctx); err != nil {
			b.Fatalf("apply rules: %v", err)
		}
	}
}
