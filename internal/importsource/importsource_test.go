package importsource

import (
	"context"
	"strings"
	"testing"

	"github.com/thedavidweng/money/internal/core"
)

func TestNormalizeHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"lowercase", []string{"date", "amount"}, []string{"date", "amount"}},
		{"uppercase", []string{"DATE", "AMOUNT"}, []string{"date", "amount"}},
		{"mixed case", []string{"Date", "Account Name"}, []string{"date", "account_name"}},
		{"spaces to underscores", []string{"first name", "last name"}, []string{"first_name", "last_name"}},
		{"leading/trailing spaces", []string{"  date  ", " amount "}, []string{"date", "amount"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeHeaders(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestGetCell(t *testing.T) {
	colIndex := map[string]int{"date": 0, "amount": 1, "name": 2}
	row := []string{"2024-01-01", " 100.50 ", "Coffee Shop"}

	tests := []struct {
		col      string
		expected string
	}{
		{"date", "2024-01-01"},
		{"amount", "100.50"},
		{"name", "Coffee Shop"},
		{"missing", ""},
	}
	for _, tt := range tests {
		t.Run(tt.col, func(t *testing.T) {
			got := getCell(row, colIndex, tt.col)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}

	// Index out of bounds
	oobRow := []string{"a"}
	got := getCell(oobRow, map[string]int{"x": 5}, "x")
	if got != "" {
		t.Errorf("out-of-bounds: got %q, want empty", got)
	}
}

func TestParseCSVAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"positive", "100.50", 10050, false},
		{"negative", "-42.99", -4299, false},
		{"dollar sign", "$1,234.56", 123456, false},
		{"pound sign", "£50.00", 5000, false},
		{"euro sign", "€75.25", 7525, false},
		{"thousands separator", "1,000.00", 100000, false},
		{"whitespace", "  100.00  ", 10000, false},
		{"zero", "0", 0, false},
		{"empty", "", 0, true},
		{"invalid", "abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCSVAmount(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHashCSVRow(t *testing.T) {
	row1 := []string{"2024-01-01", "100.00", "Coffee"}
	row2 := []string{"2024-01-01", "100.00", "Coffee"}
	row3 := []string{"2024-01-01", "100.00", "Tea"}

	h1 := hashCSVRow(row1)
	h2 := hashCSVRow(row2)
	h3 := hashCSVRow(row3)

	if h1 != h2 {
		t.Errorf("same rows should produce same hash: %s != %s", h1, h2)
	}
	if h1 == h3 {
		t.Errorf("different rows should produce different hash: %s == %s", h1, h3)
	}
	if len(h1) != 16 {
		t.Errorf("hash length = %d, want 16", len(h1))
	}
}

// mockImportStore implements ImportStore for testing.
type mockImportStore struct {
	accounts     []core.Account
	transactions []core.Transaction
	inserted     bool
	dupIDs       []string
}

func (m *mockImportStore) UpsertImportedAccount(ctx context.Context, account core.Account) error {
	m.accounts = append(m.accounts, account)
	return nil
}

func (m *mockImportStore) UpsertImportedTransaction(ctx context.Context, tx core.Transaction, sourceRowHash string) (bool, []string, error) {
	m.transactions = append(m.transactions, tx)
	return m.inserted, m.dupIDs, nil
}

func TestCSVImporter_Name(t *testing.T) {
	c := &CSVImporter{}
	if c.Name() != "csv" {
		t.Errorf("Name() = %q, want %q", c.Name(), "csv")
	}
}

func TestCSVImporter_Import_Success(t *testing.T) {
	csv := "date,amount,name,currency\n2024-01-01,100.50,Coffee,USD\n2024-01-02,-42.99,Lunch,USD\n"
	store := &mockImportStore{inserted: true}
	c := &CSVImporter{}

	result, err := c.Import(context.Background(), store, "batch1", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccountsImported != 1 {
		t.Errorf("AccountsImported = %d, want 1", result.AccountsImported)
	}
	if result.TransactionsImported != 2 {
		t.Errorf("TransactionsImported = %d, want 2", result.TransactionsImported)
	}
	if result.DuplicatesSkipped != 0 {
		t.Errorf("DuplicatesSkipped = %d, want 0", result.DuplicatesSkipped)
	}
	if len(store.accounts) != 1 {
		t.Fatalf("accounts stored = %d, want 1", len(store.accounts))
	}
	if store.accounts[0].Name != "CSV Import" {
		t.Errorf("account name = %q, want %q", store.accounts[0].Name, "CSV Import")
	}
}

func TestCSVImporter_Import_CustomAccountName(t *testing.T) {
	csv := "date,amount,name,currency,account_name\n2024-01-01,10.00,Coffee,USD,Checking\n"
	store := &mockImportStore{inserted: true}
	c := &CSVImporter{}

	result, err := c.Import(context.Background(), store, "batch1", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.accounts[0].Name != "Checking" {
		t.Errorf("account name = %q, want %q", store.accounts[0].Name, "Checking")
	}
	if result.AccountsImported != 1 {
		t.Errorf("AccountsImported = %d, want 1", result.AccountsImported)
	}
}

func TestCSVImporter_Import_Duplicates(t *testing.T) {
	csv := "date,amount,name,currency\n2024-01-01,100.00,Coffee,USD\n"
	store := &mockImportStore{inserted: false}
	c := &CSVImporter{}

	result, err := c.Import(context.Background(), store, "batch1", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TransactionsImported != 0 {
		t.Errorf("TransactionsImported = %d, want 0", result.TransactionsImported)
	}
	if result.DuplicatesSkipped != 1 {
		t.Errorf("DuplicatesSkipped = %d, want 1", result.DuplicatesSkipped)
	}
}

func TestCSVImporter_Import_EmptyCSV(t *testing.T) {
	csv := "date,amount,name,currency\n"
	store := &mockImportStore{}
	c := &CSVImporter{}

	_, err := c.Import(context.Background(), store, "batch1", strings.NewReader(csv))
	if err == nil {
		t.Fatal("expected error for empty CSV")
	}
	importErr, ok := err.(ImportError)
	if !ok {
		t.Fatalf("err = %T, want ImportError", err)
	}
	if importErr.Code != "EMPTY_CSV" {
		t.Errorf("code = %q, want %q", importErr.Code, "EMPTY_CSV")
	}
}

func TestCSVImporter_Import_MissingColumn(t *testing.T) {
	csv := "date,amount\n2024-01-01,100.00\n"
	store := &mockImportStore{}
	c := &CSVImporter{}

	_, err := c.Import(context.Background(), store, "batch1", strings.NewReader(csv))
	if err == nil {
		t.Fatal("expected error for missing column")
	}
	importErr, ok := err.(ImportError)
	if !ok {
		t.Fatalf("err = %T, want ImportError", err)
	}
	if importErr.Code != "MISSING_COLUMN" {
		t.Errorf("code = %q, want %q", importErr.Code, "MISSING_COLUMN")
	}
}

func TestCSVImporter_Import_MultipleAccounts(t *testing.T) {
	csv := "date,amount,name,currency,account_name\n2024-01-01,10.00,Coffee,USD,Checking\n2024-01-02,20.00,Lunch,USD,Savings\n"
	store := &mockImportStore{inserted: true}
	c := &CSVImporter{}

	result, err := c.Import(context.Background(), store, "batch1", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccountsImported != 2 {
		t.Errorf("AccountsImported = %d, want 2", result.AccountsImported)
	}
	if result.TransactionsImported != 2 {
		t.Errorf("TransactionsImported = %d, want 2", result.TransactionsImported)
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	if len(r.Names()) != 0 {
		t.Errorf("empty registry should have 0 names, got %d", len(r.Names()))
	}

	src := &CSVImporter{}
	r.Register(src)

	got, ok := r.Get("csv")
	if !ok {
		t.Fatal("expected to find 'csv' in registry")
	}
	if got.Name() != "csv" {
		t.Errorf("got.Name() = %q, want %q", got.Name(), "csv")
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("expected 'nonexistent' to not be found")
	}

	names := r.Names()
	if len(names) != 1 || names[0] != "csv" {
		t.Errorf("Names() = %v, want [csv]", names)
	}
}

func TestDefaultRegistry(t *testing.T) {
	r := DefaultRegistry()
	names := r.Names()
	if len(names) != 2 {
		t.Fatalf("DefaultRegistry should have 2 sources, got %d", len(names))
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["csv"] {
		t.Error("DefaultRegistry should contain 'csv'")
	}
	if !nameSet["monarch"] {
		t.Error("DefaultRegistry should contain 'monarch'")
	}
}

func TestImportError(t *testing.T) {
	e := ImportError{Code: "TEST", Message: "something went wrong"}
	want := "import error TEST: something went wrong"
	if e.Error() != want {
		t.Errorf("Error() = %q, want %q", e.Error(), want)
	}
}
