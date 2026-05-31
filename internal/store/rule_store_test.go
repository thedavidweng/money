package store

import (
	"context"
	"strings"
	"testing"

	"github.com/thedavidweng/money/internal/core"
)

func TestSQLiteStoreRuleEngineAppliesCategoryByMerchantContains(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	// Store a linked item and a transaction with a known merchant.
	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_rule", Name: "Rule Bank", Provider: "plaid", ProviderInstitutionID: "ins_rule"},
		Item: LinkedItem{
			ID:                     "pi_rule",
			Provider:               "plaid",
			InstitutionID:          "inst_rule",
			ProviderExternalItemID: "item_rule",
			EncryptedAccessToken:   []byte("token"),
			Status:                 "active",
			Products:               []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}
	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID:    "pi_rule",
		ProviderAccountID: "acc_rule",
		Name:              "Checking",
		Type:              "depository",
		Currency:          "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID:        "pi_rule",
		ProviderTransactionID: "tx_starbucks",
		ProviderAccountID:     "acc_rule",
		Date:                  "2026-05-01",
		AmountMinorUnits:      -550,
		Name:                  "STARBUCKS #12345",
		MerchantName:          "Starbucks",
		Currency:              "USD",
	}); err != nil {
		t.Fatalf("upsert transaction: %v", err)
	}
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID:        "pi_rule",
		ProviderTransactionID: "tx_grocery",
		ProviderAccountID:     "acc_rule",
		Date:                  "2026-05-02",
		AmountMinorUnits:      -4500,
		Name:                  "WHOLE FOODS MARKET",
		MerchantName:          "Whole Foods",
		Currency:              "USD",
	}); err != nil {
		t.Fatalf("upsert transaction: %v", err)
	}

	// Get a category ID from demo data.
	categories, err := db.ListCategories(ctx)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(categories) == 0 {
		t.Skip("demo has no categories")
	}
	coffeeCatID := categories[0].ID

	// Create a rule: merchant_name contains "starbucks" → set category.
	_, err = db.CreateRule(ctx, core.Rule{
		Name:           "Starbucks → Coffee",
		ConditionField: "merchant_name",
		ConditionOp:    "contains",
		ConditionValue: "starbucks",
		ActionType:     "set_category",
		ActionValue:    coffeeCatID,
		Priority:       10,
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	// Apply rules.
	result, err := db.ApplyRules(ctx)
	if err != nil {
		t.Fatalf("apply rules: %v", err)
	}
	if result.TransactionsUpdated != 1 {
		t.Fatalf("transactions updated = %d, want 1", result.TransactionsUpdated)
	}

	// Verify the Starbucks transaction got the category.
	txs, err := db.SearchTransactions(ctx, "STARBUCKS", 10)
	if err != nil {
		t.Fatalf("search transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("search results = %d, want 1", len(txs))
	}
	if txs[0].CategoryID == nil || *txs[0].CategoryID != coffeeCatID {
		t.Fatalf("starbucks category = %v, want %q", txs[0].CategoryID, coffeeCatID)
	}
	if txs[0].CategorySource != "local" {
		t.Fatalf("category source = %q, want %q", txs[0].CategorySource, "local")
	}

	// Whole Foods should NOT have been categorized.
	txs2, err := db.SearchTransactions(ctx, "WHOLE FOODS", 10)
	if err != nil {
		t.Fatalf("search whole foods: %v", err)
	}
	if len(txs2) != 1 {
		t.Fatalf("whole foods results = %d, want 1", len(txs2))
	}
	if txs2[0].CategoryID != nil {
		t.Fatalf("whole foods should not be categorized, got %v", txs2[0].CategoryID)
	}
}

func TestSQLiteStoreRuleEnginePriorityHighestWins(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_pri", Name: "Pri Bank", Provider: "plaid", ProviderInstitutionID: "ins_pri"},
		Item: LinkedItem{
			ID: "pi_pri", Provider: "plaid", InstitutionID: "inst_pri",
			ProviderExternalItemID: "item_pri", EncryptedAccessToken: []byte("tok"),
			Status: "active", Products: []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}
	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID: "pi_pri", ProviderAccountID: "acc_pri",
		Name: "Checking", Type: "depository", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID: "pi_pri", ProviderTransactionID: "tx_pri",
		ProviderAccountID: "acc_pri", Date: "2026-05-01",
		AmountMinorUnits: -1000, Name: "UBER EATS", MerchantName: "Uber Eats", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert transaction: %v", err)
	}

	categories, err := db.ListCategories(ctx)
	if err != nil || len(categories) < 2 {
		t.Skip("need at least 2 demo categories")
	}
	catA, catB := categories[0].ID, categories[1].ID

	// Low priority rule: name contains "uber" → catA.
	_, err = db.CreateRule(ctx, core.Rule{
		Name: "Uber low", ConditionField: "name", ConditionOp: "contains",
		ConditionValue: "uber", ActionType: "set_category", ActionValue: catA,
		Priority: 1, Enabled: true,
	})
	if err != nil {
		t.Fatalf("create low rule: %v", err)
	}
	// High priority rule: merchant_name contains "uber eats" → catB.
	_, err = db.CreateRule(ctx, core.Rule{
		Name: "Uber Eats high", ConditionField: "merchant_name", ConditionOp: "contains",
		ConditionValue: "uber eats", ActionType: "set_category", ActionValue: catB,
		Priority: 100, Enabled: true,
	})
	if err != nil {
		t.Fatalf("create high rule: %v", err)
	}

	result, err := db.ApplyRules(ctx)
	if err != nil {
		t.Fatalf("apply rules: %v", err)
	}
	if result.TransactionsUpdated != 1 {
		t.Fatalf("transactions updated = %d, want 1", result.TransactionsUpdated)
	}

	txs, err := db.SearchTransactions(ctx, "UBER EATS", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("results = %d, want 1", len(txs))
	}
	// Highest priority rule should win.
	if txs[0].CategoryID == nil || *txs[0].CategoryID != catB {
		t.Fatalf("uber eats category = %v, want %q (high priority)", txs[0].CategoryID, catB)
	}
}

func TestSQLiteStoreRuleEngineSetNoteAction(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_note", Name: "Note Bank", Provider: "plaid", ProviderInstitutionID: "ins_note"},
		Item: LinkedItem{
			ID: "pi_note", Provider: "plaid", InstitutionID: "inst_note",
			ProviderExternalItemID: "item_note", EncryptedAccessToken: []byte("tok"),
			Status: "active", Products: []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}
	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID: "pi_note", ProviderAccountID: "acc_note",
		Name: "Checking", Type: "depository", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID: "pi_note", ProviderTransactionID: "tx_note",
		ProviderAccountID: "acc_note", Date: "2026-05-01",
		AmountMinorUnits: -2500, Name: "AMAZON MARKETPLACE", MerchantName: "Amazon", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert transaction: %v", err)
	}

	_, err = db.CreateRule(ctx, core.Rule{
		Name: "Amazon note", ConditionField: "merchant_name", ConditionOp: "equals",
		ConditionValue: "Amazon", ActionType: "set_note", ActionValue: "online shopping",
		Priority: 5, Enabled: true,
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	result, err := db.ApplyRules(ctx)
	if err != nil {
		t.Fatalf("apply rules: %v", err)
	}
	if result.TransactionsUpdated != 1 {
		t.Fatalf("transactions updated = %d, want 1", result.TransactionsUpdated)
	}

	// Verify via raw SQL since Transaction.Note is not exposed in search.
	db2 := db.db
	var note string
	if err := db2.QueryRowContext(ctx, `SELECT note FROM transactions WHERE provider_transaction_id = ?`, "tx_note").Scan(&note); err != nil {
		t.Fatalf("query note: %v", err)
	}
	if note != "online shopping" {
		t.Fatalf("note = %q, want %q", note, "online shopping")
	}
}

func TestSQLiteStoreRuleEngineDisabledRulesSkipped(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_dis", Name: "Dis Bank", Provider: "plaid", ProviderInstitutionID: "ins_dis"},
		Item: LinkedItem{
			ID: "pi_dis", Provider: "plaid", InstitutionID: "inst_dis",
			ProviderExternalItemID: "item_dis", EncryptedAccessToken: []byte("tok"),
			Status: "active", Products: []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}
	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID: "pi_dis", ProviderAccountID: "acc_dis",
		Name: "Checking", Type: "depository", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID: "pi_dis", ProviderTransactionID: "tx_dis",
		ProviderAccountID: "acc_dis", Date: "2026-05-01",
		AmountMinorUnits: -1500, Name: "NETFLIX", MerchantName: "Netflix", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert transaction: %v", err)
	}

	categories, err := db.ListCategories(ctx)
	if err != nil || len(categories) == 0 {
		t.Skip("demo has no categories")
	}

	// Create a disabled rule.
	_, err = db.CreateRule(ctx, core.Rule{
		Name: "Netflix disabled", ConditionField: "merchant_name", ConditionOp: "equals",
		ConditionValue: "Netflix", ActionType: "set_category", ActionValue: categories[0].ID,
		Priority: 5, Enabled: false,
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	result, err := db.ApplyRules(ctx)
	if err != nil {
		t.Fatalf("apply rules: %v", err)
	}
	if result.TransactionsUpdated != 0 {
		t.Fatalf("transactions updated = %d, want 0 (disabled rule)", result.TransactionsUpdated)
	}
}

func TestSQLiteStoreRuleCRUDAndListOnlyEnabled(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	r1, err := db.CreateRule(ctx, core.Rule{
		Name: "Rule A", ConditionField: "name", ConditionOp: "contains",
		ConditionValue: "test", ActionType: "set_note", ActionValue: "a",
		Priority: 1, Enabled: true,
	})
	if err != nil {
		t.Fatalf("create rule 1: %v", err)
	}
	r2, err := db.CreateRule(ctx, core.Rule{
		Name: "Rule B", ConditionField: "name", ConditionOp: "contains",
		ConditionValue: "test", ActionType: "set_note", ActionValue: "b",
		Priority: 2, Enabled: false,
	})
	if err != nil {
		t.Fatalf("create rule 2: %v", err)
	}

	// ListRules only returns enabled rules.
	rules, err := db.ListRules(ctx)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	for _, r := range rules {
		if r.ID == r2.ID {
			t.Fatal("disabled rule should not appear in ListRules")
		}
	}
	found := false
	for _, r := range rules {
		if r.ID == r1.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("enabled rule should appear in ListRules")
	}

	// Update rule.
	_, err = db.UpdateRule(ctx, core.Rule{
		ID: r1.ID, Name: "Rule A updated", ConditionField: "name", ConditionOp: "equals",
		ConditionValue: "test", ActionType: "set_note", ActionValue: "updated",
		Priority: 10, Enabled: true,
	})
	if err != nil {
		t.Fatalf("update rule: %v", err)
	}

	// Delete rules.
	if err := db.DeleteRule(ctx, r1.ID); err != nil {
		t.Fatalf("delete rule 1: %v", err)
	}
	if err := db.DeleteRule(ctx, r2.ID); err != nil {
		t.Fatalf("delete rule 2: %v", err)
	}

	rules2, err := db.ListRules(ctx)
	if err != nil {
		t.Fatalf("list rules after delete: %v", err)
	}
	for _, r := range rules2 {
		if r.ID == r1.ID || r.ID == r2.ID {
			t.Fatal("deleted rule still appears in list")
		}
	}
}

func TestSQLiteStoreApplyRulesMatchesCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_ci", Name: "CI Bank", Provider: "plaid", ProviderInstitutionID: "ins_ci"},
		Item: LinkedItem{
			ID: "pi_ci", Provider: "plaid", InstitutionID: "inst_ci",
			ProviderExternalItemID: "item_ci", EncryptedAccessToken: []byte("tok"),
			Status: "active", Products: []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}
	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID: "pi_ci", ProviderAccountID: "acc_ci",
		Name: "Checking", Type: "depository", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID: "pi_ci", ProviderTransactionID: "tx_ci",
		ProviderAccountID: "acc_ci", Date: "2026-05-01",
		AmountMinorUnits: -800, Name: "Spotify Premium", MerchantName: "Spotify", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert transaction: %v", err)
	}

	categories, err := db.ListCategories(ctx)
	if err != nil || len(categories) == 0 {
		t.Skip("demo has no categories")
	}

	// Rule with lowercase, transaction has mixed case.
	_, err = db.CreateRule(ctx, core.Rule{
		Name: "Spotify rule", ConditionField: "merchant_name", ConditionOp: "contains",
		ConditionValue: "spotify", ActionType: "set_category", ActionValue: categories[0].ID,
		Priority: 5, Enabled: true,
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	result, err := db.ApplyRules(ctx)
	if err != nil {
		t.Fatalf("apply rules: %v", err)
	}
	if result.TransactionsUpdated != 1 {
		t.Fatalf("transactions updated = %d, want 1 (case-insensitive match)", result.TransactionsUpdated)
	}
}

func TestSQLiteStoreApplyRulesReappliesOnSubsequentRuns(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_idem", Name: "Idem Bank", Provider: "plaid", ProviderInstitutionID: "ins_idem"},
		Item: LinkedItem{
			ID: "pi_idem", Provider: "plaid", InstitutionID: "inst_idem",
			ProviderExternalItemID: "item_idem", EncryptedAccessToken: []byte("tok"),
			Status: "active", Products: []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}
	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID: "pi_idem", ProviderAccountID: "acc_idem",
		Name: "Checking", Type: "depository", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID: "pi_idem", ProviderTransactionID: "tx_idem",
		ProviderAccountID: "acc_idem", Date: "2026-05-01",
		AmountMinorUnits: -1200, Name: "LYFT RIDE", MerchantName: "Lyft", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert transaction: %v", err)
	}

	categories, err := db.ListCategories(ctx)
	if err != nil || len(categories) == 0 {
		t.Skip("demo has no categories")
	}

	_, err = db.CreateRule(ctx, core.Rule{
		Name: "Lyft rule", ConditionField: "merchant_name", ConditionOp: "equals",
		ConditionValue: "Lyft", ActionType: "set_category", ActionValue: categories[0].ID,
		Priority: 5, Enabled: true,
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	// ApplyRules re-matches every run (not idempotent by design).
	r1, err := db.ApplyRules(ctx)
	if err != nil {
		t.Fatalf("apply rules 1: %v", err)
	}
	r2, err := db.ApplyRules(ctx)
	if err != nil {
		t.Fatalf("apply rules 2: %v", err)
	}
	if r1.TransactionsUpdated != 1 {
		t.Fatalf("first apply = %d, want 1", r1.TransactionsUpdated)
	}
	if r2.TransactionsUpdated != 1 {
		t.Fatalf("second apply = %d, want 1 (re-applies by design)", r2.TransactionsUpdated)
	}

	// Verify category is still correct after re-apply.
	txs, err := db.SearchTransactions(ctx, "LYFT", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(txs) != 1 || txs[0].CategoryID == nil || *txs[0].CategoryID != categories[0].ID {
		t.Fatalf("lyft category = %v, want %q", txs[0].CategoryID, categories[0].ID)
	}
}

func TestSQLiteStoreApplyRulesUsesLocalCategorySource(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	if err := db.StoreLinkedProviderItem(ctx, LinkedProviderItem{
		Institution: LinkedInstitution{ID: "inst_cs", Name: "CS Bank", Provider: "plaid", ProviderInstitutionID: "ins_cs"},
		Item: LinkedItem{
			ID: "pi_cs", Provider: "plaid", InstitutionID: "inst_cs",
			ProviderExternalItemID: "item_cs", EncryptedAccessToken: []byte("tok"),
			Status: "active", Products: []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}
	if err := db.UpsertAccount(ctx, core.FinancialAccount{
		ProviderItemID: "pi_cs", ProviderAccountID: "acc_cs",
		Name: "Checking", Type: "depository", Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}
	providerCat := "Food & Drink"
	if err := db.UpsertTransaction(ctx, core.ProviderTransaction{
		ProviderItemID: "pi_cs", ProviderTransactionID: "tx_cs",
		ProviderAccountID: "acc_cs", Date: "2026-05-01",
		AmountMinorUnits: -900, Name: "CHIPOTLE", MerchantName: "Chipotle",
		ProviderCategory: &providerCat, Currency: "USD",
	}); err != nil {
		t.Fatalf("upsert transaction: %v", err)
	}

	categories, err := db.ListCategories(ctx)
	if err != nil || len(categories) == 0 {
		t.Skip("demo has no categories")
	}

	_, err = db.CreateRule(ctx, core.Rule{
		Name: "Chipotle", ConditionField: "merchant_name", ConditionOp: "contains",
		ConditionValue: "chipotle", ActionType: "set_category", ActionValue: categories[0].ID,
		Priority: 5, Enabled: true,
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	result, err := db.ApplyRules(ctx)
	if err != nil {
		t.Fatalf("apply rules: %v", err)
	}
	if result.TransactionsUpdated != 1 {
		t.Fatalf("transactions updated = %d, want 1", result.TransactionsUpdated)
	}

	txs, err := db.SearchTransactions(ctx, "CHIPOTLE", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("results = %d, want 1", len(txs))
	}
	// category_source should be "local" (set by rule), not "provider".
	if txs[0].CategorySource != "local" {
		t.Fatalf("category source = %q, want %q", txs[0].CategorySource, "local")
	}
	// Provider category should still be preserved.
	if txs[0].ProviderCategory == nil || !strings.Contains(*txs[0].ProviderCategory, "Food") {
		t.Fatalf("provider category = %v, want Food & Drink", txs[0].ProviderCategory)
	}
}
