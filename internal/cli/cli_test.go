package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDemoAccountsListJSONUsesEnvelopeAndDoesNotNeedConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run(context.Background(), []string{"demo", "accounts", "list", "--json"}, nil, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Accounts []struct {
				ID     string `json:"id"`
				Source struct {
					Kind                   string  `json:"kind"`
					Provider               *string `json:"provider"`
					ProviderItemID         *string `json:"provider_item_id"`
					ProviderExternalItemID *string `json:"provider_external_item_id"`
					InstitutionID          *string `json:"institution_id"`
					ProviderAccountID      *string `json:"provider_account_id"`
					ProviderTransactionID  *string `json:"provider_transaction_id"`
					ImportSourceID         *string `json:"import_source_id"`
					ImportBatchID          *string `json:"import_batch_id"`
				} `json:"source"`
			} `json:"accounts"`
		} `json:"data"`
		Meta struct {
			Command       string `json:"command"`
			SchemaVersion string `json:"schema_version"`
			Demo          bool   `json:"demo"`
		} `json:"meta"`
		Warnings []any `json:"warnings"`
		Errors   []any `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON envelope: %v\n%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatalf("ok = false, errors = %#v", envelope.Errors)
	}
	if envelope.Meta.Command != "accounts.list" {
		t.Fatalf("command = %q, want accounts.list", envelope.Meta.Command)
	}
	if !envelope.Meta.Demo {
		t.Fatal("meta.demo = false, want true")
	}
	if len(envelope.Data.Accounts) < 3 {
		t.Fatalf("accounts length = %d, want deterministic demo fixtures", len(envelope.Data.Accounts))
	}
	if envelope.Data.Accounts[0].Source.ProviderTransactionID != nil {
		t.Fatalf("account source provider_transaction_id = %v, want null", *envelope.Data.Accounts[0].Source.ProviderTransactionID)
	}
}

func TestDemoTransactionSearchJSONIncludesPendingReviewTagsAndNote(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run(context.Background(), []string{"demo", "transactions", "search", "coffee", "--json"}, nil, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Transactions []struct {
				ID          string   `json:"id"`
				Pending     bool     `json:"pending"`
				NeedsReview bool     `json:"needs_review"`
				Note        *string  `json:"note"`
				TagIDs      []string `json:"tag_ids"`
				Tags        []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"tags"`
				CategorySource string `json:"category_source"`
			} `json:"transactions"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
			Demo    bool   `json:"demo"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON envelope: %v\n%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if envelope.Meta.Command != "transactions.search" {
		t.Fatalf("command = %q, want transactions.search", envelope.Meta.Command)
	}
	if len(envelope.Data.Transactions) != 1 {
		t.Fatalf("transactions length = %d, want 1: %s", len(envelope.Data.Transactions), stdout.String())
	}
	tx := envelope.Data.Transactions[0]
	if !tx.Pending || !tx.NeedsReview {
		t.Fatalf("pending/review = %v/%v, want true/true", tx.Pending, tx.NeedsReview)
	}
	if tx.Note == nil || *tx.Note == "" {
		t.Fatal("note is empty, want local annotation")
	}
	if len(tx.TagIDs) != len(tx.Tags) || len(tx.Tags) == 0 {
		t.Fatalf("tag ids/tags mismatch: %#v %#v", tx.TagIDs, tx.Tags)
	}
	if tx.CategorySource != "provider" {
		t.Fatalf("category_source = %q, want provider", tx.CategorySource)
	}
}

func TestDemoReadCommandsReturnObjectWrappedJSONCollections(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		command   string
		dataField string
	}{
		{name: "transactions list", args: []string{"demo", "transactions", "list", "--json"}, command: "transactions.list", dataField: "transactions"},
		{name: "tx list alias", args: []string{"demo", "tx", "list", "--json"}, command: "transactions.list", dataField: "transactions"},
		{name: "categories list", args: []string{"demo", "categories", "list", "--json"}, command: "categories.list", dataField: "categories"},
		{name: "tags list", args: []string{"demo", "tags", "list", "--json"}, command: "tags.list", dataField: "tags"},
		{name: "recurring list", args: []string{"demo", "recurring", "list", "--json"}, command: "recurring.list", dataField: "recurring"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := Run(context.Background(), tt.args, nil, &stdout, &stderr)
			if exitCode != 0 {
				t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
			}
			var envelope struct {
				OK   bool                       `json:"ok"`
				Data map[string]json.RawMessage `json:"data"`
				Meta struct {
					Command string `json:"command"`
					Demo    bool   `json:"demo"`
				} `json:"meta"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
				t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
			}
			if !envelope.OK || !envelope.Meta.Demo || envelope.Meta.Command != tt.command {
				t.Fatalf("bad envelope metadata: %#v", envelope.Meta)
			}
			raw, ok := envelope.Data[tt.dataField]
			if !ok {
				t.Fatalf("data.%s missing from %s", tt.dataField, stdout.String())
			}
			var values []any
			if err := json.Unmarshal(raw, &values); err != nil {
				t.Fatalf("data.%s is not an array: %v", tt.dataField, err)
			}
			if len(values) == 0 {
				t.Fatalf("data.%s is empty, want demo fixtures", tt.dataField)
			}
		})
	}
}

func TestDemoItemsGetHumanModeDoesNotEmitJSONEnvelope(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run(context.Background(), []string{"demo", "items", "get", "pi_demo_plaid"}, nil, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable output:\n%s", stdout.String())
	}
	for _, want := range []string{"pi_demo_plaid", "plaid", "inst_demo_chase", "active", "transactions"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestDemoTransactionsListJSONSupportsFiltersAndPaginationMetadata(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "transactions", "list", "--json",
		"--merchant", "Coffee",
		"--pending", "true",
		"--needs-review", "true",
		"--date-from", "2026-05-01",
		"--date-to", "2026-05-31",
		"--limit", "1",
		"--offset", "0",
	}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", exitCode, stderr.String(), stdout.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Transactions []struct {
				Name        string `json:"name"`
				Pending     bool   `json:"pending"`
				NeedsReview bool   `json:"needs_review"`
			} `json:"transactions"`
		} `json:"data"`
		Meta struct {
			Pagination *struct {
				Limit   int  `json:"limit"`
				Offset  int  `json:"offset"`
				HasMore bool `json:"has_more"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if len(envelope.Data.Transactions) != 1 || envelope.Data.Transactions[0].Name != "Blue Bottle Coffee" {
		t.Fatalf("transactions = %#v", envelope.Data.Transactions)
	}
	if envelope.Meta.Pagination == nil || envelope.Meta.Pagination.Limit != 1 || envelope.Meta.Pagination.Offset != 0 {
		t.Fatalf("pagination = %#v", envelope.Meta.Pagination)
	}
}

func TestDemoTransactionsSearchJSONSupportsLimitAndPaginationMetadata(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "transactions", "search", "e", "--json", "--limit", "1"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", exitCode, stderr.String(), stdout.String())
	}

	var envelope struct {
		Data struct {
			Transactions []json.RawMessage `json:"transactions"`
		} `json:"data"`
		Meta struct {
			Pagination *struct {
				Limit int `json:"limit"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if len(envelope.Data.Transactions) != 1 {
		t.Fatalf("transactions length = %d", len(envelope.Data.Transactions))
	}
	if envelope.Meta.Pagination == nil || envelope.Meta.Pagination.Limit != 1 {
		t.Fatalf("pagination = %#v", envelope.Meta.Pagination)
	}
}

func TestManualAccountDryRunJSONDoesNotPersistInDemoRuntime(t *testing.T) {
	var dryRunOut, dryRunErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "accounts", "create-manual",
		"--name", "Emergency Cash",
		"--type", "depository",
		"--balance", "123.45",
		"--currency", "USD",
		"--dry-run",
		"--json",
	}, nil, &dryRunOut, &dryRunErr)
	if exitCode != 0 {
		t.Fatalf("dry-run exit code = %d, stderr = %s", exitCode, dryRunErr.String())
	}

	var dryRunEnvelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Plan struct {
				WillWrite         bool   `json:"will_write"`
				AccountName       string `json:"account_name"`
				SignedBalance     string `json:"signed_balance"`
				FinancialPosition string `json:"financial_position"`
			} `json:"plan"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunOut.Bytes(), &dryRunEnvelope); err != nil {
		t.Fatalf("dry-run stdout is not JSON: %v\n%s", err, dryRunOut.String())
	}
	if dryRunEnvelope.Data.Plan.WillWrite {
		t.Fatal("dry-run plan will_write = true")
	}
	if dryRunEnvelope.Data.Plan.AccountName != "Emergency Cash" || dryRunEnvelope.Data.Plan.SignedBalance != "123.45" {
		t.Fatalf("unexpected dry-run plan: %#v", dryRunEnvelope.Data.Plan)
	}

	var listOut, listErr bytes.Buffer
	exitCode = Run(context.Background(), []string{"demo", "accounts", "list", "--json"}, nil, &listOut, &listErr)
	if exitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", exitCode, listErr.String())
	}
	if strings.Contains(listOut.String(), "Emergency Cash") {
		t.Fatalf("dry-run account persisted in demo runtime: %s", listOut.String())
	}
}

func TestAccountsListJSONReadsConfiguredEncryptedStore(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	envPath := filepath.Join(dir, ".env")
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if err := os.WriteFile(configPath, []byte(`
database:
  path: ./money.db
  encryption_key:
    env: MONEY_DB_ENCRYPTION_KEY
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"accounts", "create-manual",
		"--name", "Local Savings",
		"--type", "depository",
		"--balance", "42.00",
		"--currency", "USD",
		"--confirm",
		"--json",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s, stdout = %s", exitCode, createErr.String(), createOut.String())
	}

	var listOut, listErr bytes.Buffer
	exitCode = Run(context.Background(), []string{"--config", configPath, "accounts", "list", "--json"}, nil, &listOut, &listErr)
	if exitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %s, stdout = %s", exitCode, listErr.String(), listOut.String())
	}
	if !strings.Contains(listOut.String(), "Local Savings") {
		t.Fatalf("configured store account missing from list: %s", listOut.String())
	}
	if strings.Contains(listOut.String(), "MONEY_DB_ENCRYPTION_KEY") {
		t.Fatalf("JSON output leaked secret env name unexpectedly: %s", listOut.String())
	}
}
