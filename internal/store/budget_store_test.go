package store

import (
	"context"
	"testing"

	"github.com/thedavidweng/money/internal/core"
)

func TestSQLiteStoreBudgetCRUDLifecycleWithCategories(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create a budget.
	budget, err := db.CreateBudget(ctx, core.Budget{
		Name:      "Groceries",
		Currency:  "USD",
		Period:    "monthly",
		StartDate: "2026-01-01",
		EndDate:   "2026-12-31",
	})
	if err != nil {
		t.Fatalf("create budget: %v", err)
	}
	if budget.ID == "" {
		t.Fatal("budget ID should be auto-generated")
	}
	if budget.Name != "Groceries" {
		t.Fatalf("budget name = %q, want %q", budget.Name, "Groceries")
	}

	// Add categories to the budget.
	catFood, err := db.CreateBudgetCategory(ctx, core.BudgetCategory{
		BudgetID:        budget.ID,
		Name:            "Food",
		LimitMinorUnits: 50000,
		Currency:        "USD",
	})
	if err != nil {
		t.Fatalf("create budget category food: %v", err)
	}
	if catFood.Limit != "500.00" {
		t.Fatalf("food limit = %q, want %q", catFood.Limit, "500.00")
	}

	catTransport, err := db.CreateBudgetCategory(ctx, core.BudgetCategory{
		BudgetID:        budget.ID,
		Name:            "Transport",
		LimitMinorUnits: 20000,
		Currency:        "USD",
	})
	if err != nil {
		t.Fatalf("create budget category transport: %v", err)
	}

	// List budgets — should include categories.
	budgets, err := db.ListBudgets(ctx)
	if err != nil {
		t.Fatalf("list budgets: %v", err)
	}
	found := false
	for _, b := range budgets {
		if b.ID == budget.ID {
			found = true
			if len(b.Categories) != 2 {
				t.Fatalf("budget categories = %d, want 2", len(b.Categories))
			}
			if b.Categories[0].Name != "Food" || b.Categories[1].Name != "Transport" {
				t.Fatalf("budget categories = %v", b.Categories)
			}
		}
	}
	if !found {
		t.Fatal("created budget not found in list")
	}

	// Get single budget.
	got, err := db.GetBudget(ctx, budget.ID)
	if err != nil {
		t.Fatalf("get budget: %v", err)
	}
	if got.Name != "Groceries" || len(got.Categories) != 2 {
		t.Fatalf("get budget = %+v", got)
	}

	// Update budget name.
	updated, err := db.UpdateBudget(ctx, core.Budget{
		ID:        budget.ID,
		Name:      "Monthly Groceries",
		Currency:  "USD",
		Period:    "monthly",
		StartDate: "2026-01-01",
		EndDate:   "2026-12-31",
	})
	if err != nil {
		t.Fatalf("update budget: %v", err)
	}
	if updated.Name != "Monthly Groceries" {
		t.Fatalf("updated name = %q, want %q", updated.Name, "Monthly Groceries")
	}

	// Delete a category.
	if err := db.DeleteBudgetCategory(ctx, catTransport.ID); err != nil {
		t.Fatalf("delete budget category: %v", err)
	}
	got2, err := db.GetBudget(ctx, budget.ID)
	if err != nil {
		t.Fatalf("get budget after category delete: %v", err)
	}
	if len(got2.Categories) != 1 || got2.Categories[0].ID != catFood.ID {
		t.Fatalf("budget categories after delete = %v", got2.Categories)
	}

	// Delete budget (cascades to remaining category).
	if err := db.DeleteBudget(ctx, budget.ID); err != nil {
		t.Fatalf("delete budget: %v", err)
	}
	budgets2, err := db.ListBudgets(ctx)
	if err != nil {
		t.Fatalf("list budgets after delete: %v", err)
	}
	for _, b := range budgets2 {
		if b.ID == budget.ID {
			t.Fatal("deleted budget still appears in list")
		}
	}

	// GetBudget for deleted budget should error.
	_, err = db.GetBudget(ctx, budget.ID)
	if err == nil {
		t.Fatal("expected error getting deleted budget")
	}
}

func TestSQLiteStoreBudgetCategoryWithLinkedCategory(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	categories, err := db.ListCategories(ctx)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(categories) == 0 {
		t.Skip("demo has no categories")
	}

	budget, err := db.CreateBudget(ctx, core.Budget{
		Name:      "Dining",
		Currency:  "USD",
		Period:    "monthly",
		StartDate: "2026-01-01",
		EndDate:   "2026-12-31",
	})
	if err != nil {
		t.Fatalf("create budget: %v", err)
	}

	// Create budget category linked to a real category.
	catID := categories[0].ID
	bc, err := db.CreateBudgetCategory(ctx, core.BudgetCategory{
		BudgetID:        budget.ID,
		CategoryID:      &catID,
		Name:            categories[0].Name,
		LimitMinorUnits: 10000,
		Currency:        "USD",
	})
	if err != nil {
		t.Fatalf("create budget category: %v", err)
	}
	if bc.CategoryID == nil || *bc.CategoryID != catID {
		t.Fatalf("budget category CategoryID = %v, want %q", bc.CategoryID, catID)
	}
}
