package store

import (
	"context"
	"testing"

	"github.com/thedavidweng/money/internal/core"
)

func TestInvestmentSyncLifecycleUpsertSecuritiesHoldingsThenClear(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Link a provider item for the investment account.
	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_inv", Name: "Invest Bank", Provider: "plaid", ProviderInstitutionID: "ins_inv"},
		Item: LinkedItem{
			ID: "pi_inv", Provider: "plaid", InstitutionID: "inst_inv",
			ProviderExternalItemID: "item_inv", EncryptedAccessToken: []byte("tok"),
			Status: "active", Products: []string{"investments"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}
	// Upsert the investment account.
	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID: "pi_inv", ProviderAccountID: "acc_inv",
		Name: "Brokerage", Type: "investment", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}

	// Upsert two securities.
	tickerAAPL := "AAPL"
	if err := db.UpsertSecurity(ctx, core.InvestmentSecurity{
		SecurityID:   "sec_aapl",
		Name:         "Apple Inc",
		TickerSymbol: &tickerAAPL,
		Type:         "equity",
		ClosePrice:   185.50,
		Currency:     "USD",
	}); err != nil {
		t.Fatalf("upsert AAPL: %v", err)
	}
	if err := db.UpsertSecurity(ctx, core.InvestmentSecurity{
		SecurityID: "sec_spy",
		Name:       "SPDR S&P 500 ETF",
		Type:       "etf",
		ClosePrice: 520.75,
		Currency:   "USD",
	}); err != nil {
		t.Fatalf("upsert SPY: %v", err)
	}

	// Upsert holdings.
	costBasis := 15000.00
	if err := db.UpsertHolding(ctx, "pi_inv", core.InvestmentHolding{
		AccountID:        "acc_inv",
		SecurityID:       "sec_aapl",
		Quantity:         100,
		InstitutionPrice: 185.50,
		InstitutionValue: 18550.00,
		CostBasis:        &costBasis,
		Currency:         "USD",
	}); err != nil {
		t.Fatalf("upsert AAPL holding: %v", err)
	}
	if err := db.UpsertHolding(ctx, "pi_inv", core.InvestmentHolding{
		AccountID:        "acc_inv",
		SecurityID:       "sec_spy",
		Quantity:         50,
		InstitutionPrice: 520.75,
		InstitutionValue: 26037.50,
		Currency:         "USD",
	}); err != nil {
		t.Fatalf("upsert SPY holding: %v", err)
	}

	// List securities.
	securities, err := db.ListSecurities(ctx)
	if err != nil {
		t.Fatalf("list securities: %v", err)
	}
	foundAAPL, foundSPY := false, false
	for _, s := range securities {
		if s.SecurityID == "sec_aapl" {
			foundAAPL = true
			if s.Name != "Apple Inc" {
				t.Fatalf("AAPL name = %q", s.Name)
			}
			if s.TickerSymbol == nil || *s.TickerSymbol != "AAPL" {
				t.Fatalf("AAPL ticker = %v", s.TickerSymbol)
			}
		}
		if s.SecurityID == "sec_spy" {
			foundSPY = true
		}
	}
	if !foundAAPL || !foundSPY {
		t.Fatalf("expected both securities, foundAAPL=%v foundSPY=%v", foundAAPL, foundSPY)
	}

	// List holdings.
	holdings, err := db.ListHoldings(ctx)
	if err != nil {
		t.Fatalf("list holdings: %v", err)
	}
	if len(holdings) != 2 {
		t.Fatalf("holdings = %d, want 2", len(holdings))
	}
	// Holdings are sorted by institution_value DESC.
	if holdings[0].SecurityID != "sec_spy" {
		t.Fatalf("largest holding = %s, want sec_spy", holdings[0].SecurityID)
	}
	if holdings[1].CostBasis == nil || *holdings[1].CostBasis != 15000.00 {
		t.Fatalf("AAPL cost basis = %v, want 15000", holdings[1].CostBasis)
	}

	// Upsert again with updated price (simulates re-sync).
	if err := db.UpsertSecurity(ctx, core.InvestmentSecurity{
		SecurityID: "sec_aapl",
		Name:       "Apple Inc",
		Type:       "equity",
		ClosePrice: 190.00,
		Currency:   "USD",
	}); err != nil {
		t.Fatalf("upsert AAPL updated: %v", err)
	}
	securities2, err := db.ListSecurities(ctx)
	if err != nil {
		t.Fatalf("list securities 2: %v", err)
	}
	for _, s := range securities2 {
		if s.SecurityID == "sec_aapl" {
			if s.ClosePrice != 190.00 {
				t.Fatalf("AAPL price = %f, want 190.00", s.ClosePrice)
			}
		}
	}

	// Clear holdings for this item.
	if err := db.ClearHoldings(ctx, "pi_inv"); err != nil {
		t.Fatalf("clear holdings: %v", err)
	}
	holdings2, err := db.ListHoldings(ctx)
	if err != nil {
		t.Fatalf("list holdings after clear: %v", err)
	}
	if len(holdings2) != 0 {
		t.Fatalf("holdings after clear = %d, want 0", len(holdings2))
	}
	// Securities should still exist (clear only affects holdings).
	securities3, err := db.ListSecurities(ctx)
	if err != nil {
		t.Fatalf("list securities after clear: %v", err)
	}
	if len(securities3) != 2 {
		t.Fatalf("securities after clear = %d, want 2", len(securities3))
	}
}
