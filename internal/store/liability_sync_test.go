package store

import (
	"context"
	"testing"

	"github.com/thedavidweng/money/internal/core"
)

func TestLiabilitySyncLifecycleUpsertThenClear(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Link a provider item for the liability account.
	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_liab", Name: "Loan Bank", Provider: "plaid", ProviderInstitutionID: "ins_liab"},
		Item: LinkedItem{
			ID: "pi_liab", Provider: "plaid", InstitutionID: "inst_liab",
			ProviderExternalItemID: "item_liab", EncryptedAccessToken: []byte("tok"),
			Status: "active", Products: []string{"liabilities"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}
	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID: "pi_liab", ProviderAccountID: "acc_liab",
		Name: "Student Loan", Type: "loan", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}

	// Upsert a liability.
	origBalance := 25000.00
	lastPayment := 350.00
	apr := 5.5
	nextDue := "2026-06-01"
	if err := db.UpsertLiability(ctx, "pi_liab", core.Liability{
		AccountID:          "acc_liab",
		Type:               "student",
		CurrentBalance:     22000.00,
		OriginalBalance:    &origBalance,
		Currency:           "USD",
		Name:               "Federal Student Loan",
		LastPaymentDate:    strPtr2("2026-05-01"),
		LastPaymentAmount:  &lastPayment,
		NextPaymentDueDate: &nextDue,
		APR:                &apr,
	}); err != nil {
		t.Fatalf("upsert liability: %v", err)
	}

	// List liabilities.
	liabilities, err := db.ListLiabilities(ctx)
	if err != nil {
		t.Fatalf("list liabilities: %v", err)
	}
	if len(liabilities) != 1 {
		t.Fatalf("liabilities = %d, want 1", len(liabilities))
	}
	l := liabilities[0]
	if l.Type != "student" || l.Name != "Federal Student Loan" {
		t.Fatalf("liability = %+v", l)
	}
	if l.CurrentBalance != 22000.00 {
		t.Fatalf("balance = %f, want 22000", l.CurrentBalance)
	}
	if l.OriginalBalance == nil || *l.OriginalBalance != 25000.00 {
		t.Fatalf("original balance = %v, want 25000", l.OriginalBalance)
	}
	if l.APR == nil || *l.APR != 5.5 {
		t.Fatalf("APR = %v, want 5.5", l.APR)
	}

	// Upsert again with updated balance (simulates re-sync).
	if err := db.UpsertLiability(ctx, "pi_liab", core.Liability{
		AccountID:      "acc_liab",
		Type:           "student",
		CurrentBalance: 21650.00,
		Currency:       "USD",
		Name:           "Federal Student Loan",
	}); err != nil {
		t.Fatalf("upsert liability updated: %v", err)
	}
	liabilities2, err := db.ListLiabilities(ctx)
	if err != nil {
		t.Fatalf("list liabilities 2: %v", err)
	}
	if len(liabilities2) != 1 {
		t.Fatalf("liabilities after update = %d, want 1", len(liabilities2))
	}
	if liabilities2[0].CurrentBalance != 21650.00 {
		t.Fatalf("updated balance = %f, want 21650", liabilities2[0].CurrentBalance)
	}

	// Clear liabilities.
	if err := db.ClearLiabilities(ctx, "pi_liab"); err != nil {
		t.Fatalf("clear liabilities: %v", err)
	}
	liabilities3, err := db.ListLiabilities(ctx)
	if err != nil {
		t.Fatalf("list liabilities after clear: %v", err)
	}
	if len(liabilities3) != 0 {
		t.Fatalf("liabilities after clear = %d, want 0", len(liabilities3))
	}
}

func strPtr2(s string) *string {
	return &s
}
