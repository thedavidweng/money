package importsource

import (
	"context"
	"strings"
	"testing"
)

func TestMonarchImporter_Name(t *testing.T) {
	m := &MonarchImporter{}
	if m.Name() != "monarch" {
		t.Errorf("Name() = %q, want %q", m.Name(), "monarch")
	}
}

func TestMonarchImporter_ImportFullFlowWithAccountsAndTransactions(t *testing.T) {
	json := `{
		"accounts": [
			{"id": "m_acc1", "name": "Chase Checking", "type": "CHECKING", "subtype": "checking", "balance": 5000.50, "currency": "USD"},
			{"id": "m_acc2", "name": "Amex Gold", "type": "CREDIT_CARD", "subtype": "credit", "balance": -1200.00, "currency": "USD"}
		],
		"transactions": [
			{"id": "m_tx1", "account_id": "m_acc1", "date": "2026-05-01", "amount": -4.50, "currency": "USD", "name": "STARBUCKS", "merchant_name": "Starbucks", "category": "Coffee", "pending": false},
			{"id": "m_tx2", "account_id": "m_acc1", "date": "2026-05-02", "amount": -120.00, "currency": "USD", "name": "AMAZON", "merchant_name": "Amazon", "category": "Shopping", "pending": false},
			{"id": "m_tx3", "account_id": "m_acc2", "date": "2026-05-03", "amount": -85.00, "currency": "USD", "name": "WHOLE FOODS", "merchant_name": "Whole Foods", "category": "Groceries", "pending": true}
		]
	}`
	store := &mockImportStore{inserted: true}
	m := &MonarchImporter{}

	result, err := m.Import(context.Background(), store, "batch_monarch", strings.NewReader(json))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccountsImported != 2 {
		t.Fatalf("AccountsImported = %d, want 2", result.AccountsImported)
	}
	if result.TransactionsImported != 3 {
		t.Fatalf("TransactionsImported = %d, want 3", result.TransactionsImported)
	}
	if result.DuplicatesSkipped != 0 {
		t.Fatalf("DuplicatesSkipped = %d, want 0", result.DuplicatesSkipped)
	}

	// Verify accounts were stored with correct types.
	if len(store.accounts) != 2 {
		t.Fatalf("accounts stored = %d, want 2", len(store.accounts))
	}
	// Monarch CHECKING → depository.
	if store.accounts[0].Type != "depository" {
		t.Fatalf("account 0 type = %q, want %q", store.accounts[0].Type, "depository")
	}
	// Monarch CREDIT_CARD → credit.
	if store.accounts[1].Type != "credit" {
		t.Fatalf("account 1 type = %q, want %q", store.accounts[1].Type, "credit")
	}

	// Verify account balance conversion.
	if store.accounts[0].CurrentBalanceMinorUnits != 500050 {
		t.Fatalf("account 0 balance = %d, want 500050", store.accounts[0].CurrentBalanceMinorUnits)
	}

	// Verify transactions have correct account IDs.
	if len(store.transactions) != 3 {
		t.Fatalf("transactions stored = %d, want 3", len(store.transactions))
	}
	acc1ID := store.accounts[0].ID
	acc2ID := store.accounts[1].ID
	if store.transactions[0].AccountID != acc1ID {
		t.Fatalf("tx 0 account = %q, want %q", store.transactions[0].AccountID, acc1ID)
	}
	if store.transactions[2].AccountID != acc2ID {
		t.Fatalf("tx 2 account = %q, want %q", store.transactions[2].AccountID, acc2ID)
	}

	// Verify transaction with category.
	if store.transactions[0].CategoryName == nil || *store.transactions[0].CategoryName != "Coffee" {
		t.Fatalf("tx 0 category = %v, want %q", store.transactions[0].CategoryName, "Coffee")
	}

	// Verify pending flag.
	if !store.transactions[2].Pending {
		t.Fatal("tx 3 should be pending")
	}

	// Verify source metadata.
	if store.accounts[0].Source.Kind != "import" {
		t.Fatalf("account source kind = %q, want %q", store.accounts[0].Source.Kind, "import")
	}
	if store.accounts[0].Source.ImportSourceID == nil || *store.accounts[0].Source.ImportSourceID != "monarch" {
		t.Fatalf("account import source = %v, want monarch", store.accounts[0].Source.ImportSourceID)
	}
}

func TestMonarchImporter_ImportDuplicateTransactions(t *testing.T) {
	json := `{
		"accounts": [{"id": "m_dup_acc", "name": "Bank", "type": "CHECKING", "subtype": "", "balance": 100, "currency": "USD"}],
		"transactions": [
			{"id": "m_dup_tx", "account_id": "m_dup_acc", "date": "2026-05-01", "amount": -10.00, "currency": "USD", "name": "Test", "merchant_name": "Test", "category": "", "pending": false}
		]
	}`
	store := &mockImportStore{inserted: false} // simulate duplicate
	m := &MonarchImporter{}

	result, err := m.Import(context.Background(), store, "batch_dup", strings.NewReader(json))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TransactionsImported != 0 {
		t.Fatalf("TransactionsImported = %d, want 0", result.TransactionsImported)
	}
	if result.DuplicatesSkipped != 1 {
		t.Fatalf("DuplicatesSkipped = %d, want 1", result.DuplicatesSkipped)
	}
}

func TestMonarchImporter_ImportTransactionReferencesUnknownAccount(t *testing.T) {
	json := `{
		"accounts": [{"id": "m_known", "name": "Bank", "type": "CHECKING", "subtype": "", "balance": 100, "currency": "USD"}],
		"transactions": [
			{"id": "m_orphan", "account_id": "m_unknown", "date": "2026-05-01", "amount": -10.00, "currency": "USD", "name": "Test", "merchant_name": "Test", "category": "", "pending": false}
		]
	}`
	store := &mockImportStore{inserted: true}
	m := &MonarchImporter{}

	_, err := m.Import(context.Background(), store, "batch_orphan", strings.NewReader(json))
	if err == nil {
		t.Fatal("expected error for unknown account reference")
	}
	importErr, ok := err.(ImportError)
	if !ok {
		t.Fatalf("err = %T, want ImportError", err)
	}
	if importErr.Code != "UNKNOWN_ACCOUNT" {
		t.Fatalf("code = %q, want %q", importErr.Code, "UNKNOWN_ACCOUNT")
	}
}

func TestMonarchImporter_ImportInvalidJSON(t *testing.T) {
	store := &mockImportStore{}
	m := &MonarchImporter{}

	_, err := m.Import(context.Background(), store, "batch_bad", strings.NewReader("{invalid"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	importErr, ok := err.(ImportError)
	if !ok {
		t.Fatalf("err = %T, want ImportError", err)
	}
	if importErr.Code != "INVALID_JSON" {
		t.Fatalf("code = %q, want %q", importErr.Code, "INVALID_JSON")
	}
}

func TestMonarchImporter_TypeMapping(t *testing.T) {
	tests := []struct {
		monarch string
		want    string
	}{
		{"CHECKING", "depository"},
		{"SAVINGS", "depository"},
		{"CREDIT_CARD", "credit"},
		{"INVESTMENT", "investment"},
		{"LOAN", "loan"},
		{"MORTGAGE", "loan"},
		{"PROPERTY", "property"},
		{"VEHICLE", "vehicle"},
		{"UNKNOWN_TYPE", "other_asset"},
	}
	for _, tt := range tests {
		got := mapMonarchType(tt.monarch)
		if got != tt.want {
			t.Errorf("mapMonarchType(%q) = %q, want %q", tt.monarch, got, tt.want)
		}
	}
}

func TestMonarchImporter_HashConsistency(t *testing.T) {
	mt := monarchTransaction{
		ID: "tx1", AccountID: "acc1", Date: "2026-05-01",
		Amount: -10.50, Currency: "USD", Name: "Test", Pending: false,
	}
	h1 := hashMonarchRow(&mt)
	h2 := hashMonarchRow(&mt)
	if h1 != h2 {
		t.Errorf("same input produced different hashes: %s != %s", h1, h2)
	}

	// Different amount should produce different hash.
	mt2 := mt
	mt2.Amount = -20.00
	h3 := hashMonarchRow(&mt2)
	if h1 == h3 {
		t.Errorf("different amounts produced same hash: %s", h1)
	}
}

func TestMonarchImporter_EmptyTransactions(t *testing.T) {
	json := `{
		"accounts": [{"id": "m_empty", "name": "Bank", "type": "CHECKING", "subtype": "", "balance": 100, "currency": "USD"}],
		"transactions": []
	}`
	store := &mockImportStore{inserted: true}
	m := &MonarchImporter{}

	result, err := m.Import(context.Background(), store, "batch_empty", strings.NewReader(json))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccountsImported != 1 {
		t.Fatalf("AccountsImported = %d, want 1", result.AccountsImported)
	}
	if result.TransactionsImported != 0 {
		t.Fatalf("TransactionsImported = %d, want 0", result.TransactionsImported)
	}
}

func TestMonarchImporter_TransactionAmountConversion(t *testing.T) {
	json := `{
		"accounts": [{"id": "m_amt", "name": "Bank", "type": "CHECKING", "subtype": "", "balance": 100, "currency": "USD"}],
		"transactions": [
			{"id": "m_amt_tx", "account_id": "m_amt", "date": "2026-05-01", "amount": -1234.56, "currency": "USD", "name": "Big Purchase", "merchant_name": "Store", "category": "", "pending": false}
		]
	}`
	store := &mockImportStore{inserted: true}
	m := &MonarchImporter{}

	result, err := m.Import(context.Background(), store, "batch_amt", strings.NewReader(json))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TransactionsImported != 1 {
		t.Fatalf("TransactionsImported = %d, want 1", result.TransactionsImported)
	}
	if store.transactions[0].AmountMinorUnits != -123456 {
		t.Fatalf("amount = %d, want -123456", store.transactions[0].AmountMinorUnits)
	}
}
