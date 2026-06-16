package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestDemoBudgetsListJSONReturnsEnvelopeWithEmptyBudgets(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "budgets", "list", "--json"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Budgets []json.RawMessage `json:"budgets"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
			Demo    bool   `json:"demo"`
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
	if envelope.Meta.Command != "budgets.list" {
		t.Fatalf("command = %q, want budgets.list", envelope.Meta.Command)
	}
	if !envelope.Meta.Demo {
		t.Fatal("meta.demo = false, want true")
	}
	if len(envelope.Data.Budgets) != 0 {
		t.Fatalf("budgets length = %d, want 0 for empty demo store", len(envelope.Data.Budgets))
	}
}

func TestDemoBudgetsListHumanOutputsTable(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "budgets", "list"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable table:\n%s", stdout.String())
	}
	// Even with no budgets, the table header should appear.
	out := stdout.String()
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "PERIOD") {
		t.Fatalf("stdout missing expected table headers:\n%s", out)
	}
}

func TestDemoBudgetsListVerboseJSONIncludesVerboseFields(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "budgets", "list", "--verbose", "--json"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Budgets []json.RawMessage `json:"budgets"`
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
	if envelope.Meta.Command != "budgets.list" {
		t.Fatalf("command = %q, want budgets.list", envelope.Meta.Command)
	}
	if !envelope.Meta.Demo {
		t.Fatal("meta.demo = false, want true")
	}
}

func TestDemoBudgetsListVerboseHumanIncludesIDAndCategoryColumns(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "budgets", "list", "--verbose"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "ID") || !strings.Contains(out, "CATEGORIES") {
		t.Fatalf("stdout missing verbose columns (ID/CATEGORIES):\n%s", out)
	}
}

func TestBudgetCreateDryRunJSONReturnsPlanWithoutPersisting(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "budgets", "create",
		"--name", "Groceries",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--dry-run",
		"--json",
	}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			DryRun bool `json:"dry_run"`
			Budget struct {
				Name      string `json:"name"`
				Period    string `json:"period"`
				StartDate string `json:"start_date"`
				EndDate   string `json:"end_date"`
				Currency  string `json:"currency"`
			} `json:"budget"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
			Demo    bool   `json:"demo"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if !envelope.Data.DryRun {
		t.Fatal("dry_run = false, want true")
	}
	if envelope.Data.Budget.Name != "Groceries" {
		t.Fatalf("budget name = %q, want Groceries", envelope.Data.Budget.Name)
	}
	if envelope.Data.Budget.Period != "monthly" {
		t.Fatalf("budget period = %q, want monthly", envelope.Data.Budget.Period)
	}
	if envelope.Meta.Command != "budgets.create" {
		t.Fatalf("command = %q, want budgets.create", envelope.Meta.Command)
	}

	// Verify dry-run did not persist.
	var listOut, listErr bytes.Buffer
	exitCode = Run(context.Background(), []string{"demo", "budgets", "list", "--json"}, nil, &listOut, &listErr)
	if exitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", exitCode, listErr.String())
	}
	if strings.Contains(listOut.String(), "Groceries") {
		t.Fatalf("dry-run budget persisted in demo store: %s", listOut.String())
	}
}

func TestBudgetCreateDryRunHumanOutputsPlan(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "budgets", "create",
		"--name", "Rent",
		"--period", "yearly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--dry-run",
	}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Would create budget") {
		t.Fatalf("stdout missing dry-run message:\n%s", out)
	}
	if !strings.Contains(out, "Rent") {
		t.Fatalf("stdout missing budget name:\n%s", out)
	}
}

func TestBudgetListVerboseHumanWithRealBudgetsShowsVerboseColumns(t *testing.T) {
	configPath := writeTestConfig(t, "")

	// Create a budget so the list has data.
	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "create",
		"--name", "Groceries",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--confirm",
		"--json",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", exitCode, createErr.String())
	}

	// List with --verbose in human mode.
	var listOut, listErr bytes.Buffer
	exitCode = Run(context.Background(), []string{"--config", configPath, "budgets", "list", "--verbose"}, nil, &listOut, &listErr)
	if exitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", exitCode, listErr.String())
	}
	if json.Valid(listOut.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable:\n%s", listOut.String())
	}
	out := listOut.String()
	for _, want := range []string{"ID", "NAME", "Groceries", "monthly", "CATEGORIES"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestBudgetCreateConfirmHumanOutputsCreatedBudget(t *testing.T) {
	configPath := writeTestConfig(t, "")

	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "create",
		"--name", "Savings",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--confirm",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", exitCode, createErr.String())
	}
	if json.Valid(createOut.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable:\n%s", createOut.String())
	}
	out := createOut.String()
	if !strings.Contains(out, "Created budget") {
		t.Fatalf("stdout missing 'Created budget':\n%s", out)
	}
	if !strings.Contains(out, "Savings") {
		t.Fatalf("stdout missing budget name:\n%s", out)
	}
}

func TestBudgetDeleteHumanOutputsDeletedBudget(t *testing.T) {
	configPath := writeTestConfig(t, "")

	// Create a budget.
	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "create",
		"--name", "ToDelete",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--confirm",
		"--json",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", exitCode, createErr.String())
	}
	var created struct {
		Data struct {
			Budget struct {
				ID string `json:"id"`
			} `json:"budget"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createOut.Bytes(), &created); err != nil {
		t.Fatalf("create output is not JSON: %v", err)
	}

	// Delete in human mode.
	var delOut, delErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "delete", created.Data.Budget.ID,
	}, nil, &delOut, &delErr)
	if exitCode != 0 {
		t.Fatalf("delete exit code = %d, stderr = %s", exitCode, delErr.String())
	}
	if json.Valid(delOut.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable:\n%s", delOut.String())
	}
	out := delOut.String()
	if !strings.Contains(out, "Deleted budget") {
		t.Fatalf("stdout missing 'Deleted budget':\n%s", out)
	}
	if !strings.Contains(out, created.Data.Budget.ID) {
		t.Fatalf("stdout missing budget ID:\n%s", out)
	}
}

func TestBudgetCategoriesCreateConfirmHumanOutputsCreatedCategory(t *testing.T) {
	configPath := writeTestConfig(t, "")

	// Create a budget.
	var budgetOut, budgetErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "create",
		"--name", "Monthly",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--confirm",
		"--json",
	}, nil, &budgetOut, &budgetErr)
	if exitCode != 0 {
		t.Fatalf("budget create exit code = %d, stderr = %s", exitCode, budgetErr.String())
	}
	var budgetEnv struct {
		Data struct {
			Budget struct {
				ID string `json:"id"`
			} `json:"budget"`
		} `json:"data"`
	}
	if err := json.Unmarshal(budgetOut.Bytes(), &budgetEnv); err != nil {
		t.Fatalf("budget create output is not JSON: %v", err)
	}

	// Create category in human mode.
	var catOut, catErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "categories", "create",
		"--budget-id", budgetEnv.Data.Budget.ID,
		"--name", "Groceries",
		"--limit", "50000",
		"--confirm",
	}, nil, &catOut, &catErr)
	if exitCode != 0 {
		t.Fatalf("category create exit code = %d, stderr = %s", exitCode, catErr.String())
	}
	if json.Valid(catOut.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable:\n%s", catOut.String())
	}
	out := catOut.String()
	if !strings.Contains(out, "Created budget category") {
		t.Fatalf("stdout missing 'Created budget category':\n%s", out)
	}
	if !strings.Contains(out, "Groceries") {
		t.Fatalf("stdout missing category name:\n%s", out)
	}
}

func TestBudgetCategoriesDeleteHumanOutputsDeletedCategory(t *testing.T) {
	configPath := writeTestConfig(t, "")

	// Create a budget.
	var budgetOut, budgetErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "create",
		"--name", "Monthly",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--confirm",
		"--json",
	}, nil, &budgetOut, &budgetErr)
	if exitCode != 0 {
		t.Fatalf("budget create exit code = %d, stderr = %s", exitCode, budgetErr.String())
	}
	var budgetEnv struct {
		Data struct {
			Budget struct {
				ID string `json:"id"`
			} `json:"budget"`
		} `json:"data"`
	}
	if err := json.Unmarshal(budgetOut.Bytes(), &budgetEnv); err != nil {
		t.Fatalf("budget create output is not JSON: %v", err)
	}

	// Create category.
	var catOut, catErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "categories", "create",
		"--budget-id", budgetEnv.Data.Budget.ID,
		"--name", "Dining",
		"--limit", "30000",
		"--confirm",
		"--json",
	}, nil, &catOut, &catErr)
	if exitCode != 0 {
		t.Fatalf("category create exit code = %d, stderr = %s", exitCode, catErr.String())
	}
	var catEnv struct {
		Data struct {
			BudgetCategory struct {
				ID string `json:"id"`
			} `json:"budget_category"`
		} `json:"data"`
	}
	if err := json.Unmarshal(catOut.Bytes(), &catEnv); err != nil {
		t.Fatalf("category create output is not JSON: %v", err)
	}

	// Delete in human mode.
	var delOut, delErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "categories", "delete", catEnv.Data.BudgetCategory.ID,
	}, nil, &delOut, &delErr)
	if exitCode != 0 {
		t.Fatalf("delete exit code = %d, stderr = %s", exitCode, delErr.String())
	}
	if json.Valid(delOut.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable:\n%s", delOut.String())
	}
	out := delOut.String()
	if !strings.Contains(out, "Deleted budget category") {
		t.Fatalf("stdout missing 'Deleted budget category':\n%s", out)
	}
}

func TestBudgetGetHumanWithCategoriesShowsCategoryTable(t *testing.T) {
	configPath := writeTestConfig(t, "")

	// Create a budget.
	var budgetOut, budgetErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "create",
		"--name", "Household",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--confirm",
		"--json",
	}, nil, &budgetOut, &budgetErr)
	if exitCode != 0 {
		t.Fatalf("budget create exit code = %d, stderr = %s", exitCode, budgetErr.String())
	}
	var budgetEnv struct {
		Data struct {
			Budget struct {
				ID string `json:"id"`
			} `json:"budget"`
		} `json:"data"`
	}
	if err := json.Unmarshal(budgetOut.Bytes(), &budgetEnv); err != nil {
		t.Fatalf("budget create output is not JSON: %v", err)
	}

	// Create a category.
	var catOut, catErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "categories", "create",
		"--budget-id", budgetEnv.Data.Budget.ID,
		"--name", "Groceries",
		"--limit", "50000",
		"--confirm",
		"--json",
	}, nil, &catOut, &catErr)
	if exitCode != 0 {
		t.Fatalf("category create exit code = %d, stderr = %s", exitCode, catErr.String())
	}

	// Get budget in human mode - should show categories table.
	var getOut, getErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "get", budgetEnv.Data.Budget.ID,
	}, nil, &getOut, &getErr)
	if exitCode != 0 {
		t.Fatalf("get exit code = %d, stderr = %s", exitCode, getErr.String())
	}
	out := getOut.String()
	if !strings.Contains(out, "Categories:") {
		t.Fatalf("stdout missing 'Categories:':\n%s", out)
	}
	if !strings.Contains(out, "Groceries") {
		t.Fatalf("stdout missing category name:\n%s", out)
	}
	if !strings.Contains(out, "LIMIT") {
		t.Fatalf("stdout missing LIMIT header:\n%s", out)
	}
}

func TestBudgetCreateConfirmJSONPersistsAndReturnsCreatedBudget(t *testing.T) {
	configPath := writeTestConfig(t, "")

	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "create",
		"--name", "Groceries",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--currency", "USD",
		"--confirm",
		"--json",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s, stdout = %s", exitCode, createErr.String(), createOut.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Budget struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Period    string `json:"period"`
				StartDate string `json:"start_date"`
				EndDate   string `json:"end_date"`
				Currency  string `json:"currency"`
			} `json:"budget"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(createOut.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, createOut.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if envelope.Data.Budget.ID == "" {
		t.Fatal("created budget ID is empty")
	}
	if envelope.Data.Budget.Name != "Groceries" {
		t.Fatalf("budget name = %q, want Groceries", envelope.Data.Budget.Name)
	}
	if envelope.Meta.Command != "budgets.create" {
		t.Fatalf("command = %q, want budgets.create", envelope.Meta.Command)
	}

	// Verify it persists by listing.
	var listOut, listErr bytes.Buffer
	exitCode = Run(context.Background(), []string{"--config", configPath, "budgets", "list", "--json"}, nil, &listOut, &listErr)
	if exitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", exitCode, listErr.String())
	}
	if !strings.Contains(listOut.String(), "Groceries") {
		t.Fatalf("created budget missing from list: %s", listOut.String())
	}
}

func TestBudgetGetJSONReturnsBudgetWithID(t *testing.T) {
	configPath := writeTestConfig(t, "")

	// Create a budget first.
	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "create",
		"--name", "Entertainment",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--confirm",
		"--json",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", exitCode, createErr.String())
	}

	var created struct {
		Data struct {
			Budget struct {
				ID string `json:"id"`
			} `json:"budget"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createOut.Bytes(), &created); err != nil {
		t.Fatalf("create output is not JSON: %v", err)
	}
	budgetID := created.Data.Budget.ID
	if budgetID == "" {
		t.Fatal("created budget has empty ID")
	}

	// Get the budget.
	var getOut, getErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "get", budgetID,
		"--json",
	}, nil, &getOut, &getErr)
	if exitCode != 0 {
		t.Fatalf("get exit code = %d, stderr = %s", exitCode, getErr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Budget struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"budget"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(getOut.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, getOut.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if envelope.Data.Budget.ID != budgetID {
		t.Fatalf("budget ID = %q, want %q", envelope.Data.Budget.ID, budgetID)
	}
	if envelope.Data.Budget.Name != "Entertainment" {
		t.Fatalf("budget name = %q, want Entertainment", envelope.Data.Budget.Name)
	}
	if envelope.Meta.Command != "budgets.get" {
		t.Fatalf("command = %q, want budgets.get", envelope.Meta.Command)
	}
}

func TestBudgetGetHumanOutputsBudgetDetails(t *testing.T) {
	configPath := writeTestConfig(t, "")

	// Create a budget.
	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "create",
		"--name", "Travel",
		"--period", "yearly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--confirm",
		"--json",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", exitCode, createErr.String())
	}
	var created struct {
		Data struct {
			Budget struct {
				ID string `json:"id"`
			} `json:"budget"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createOut.Bytes(), &created); err != nil {
		t.Fatalf("create output is not JSON: %v", err)
	}

	// Get the budget in human mode.
	var getOut, getErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "get", created.Data.Budget.ID,
	}, nil, &getOut, &getErr)
	if exitCode != 0 {
		t.Fatalf("get exit code = %d, stderr = %s", exitCode, getErr.String())
	}
	if json.Valid(getOut.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable:\n%s", getOut.String())
	}
	out := getOut.String()
	for _, want := range []string{"Travel", "yearly", "2026-01-01", "2026-12-31", "USD"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestBudgetDeleteJSONRemovesBudget(t *testing.T) {
	configPath := writeTestConfig(t, "")

	// Create a budget.
	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "create",
		"--name", "Disposable",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--confirm",
		"--json",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", exitCode, createErr.String())
	}
	var created struct {
		Data struct {
			Budget struct {
				ID string `json:"id"`
			} `json:"budget"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createOut.Bytes(), &created); err != nil {
		t.Fatalf("create output is not JSON: %v", err)
	}
	budgetID := created.Data.Budget.ID

	// Delete the budget.
	var delOut, delErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "delete", budgetID,
		"--json",
	}, nil, &delOut, &delErr)
	if exitCode != 0 {
		t.Fatalf("delete exit code = %d, stderr = %s", exitCode, delErr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(delOut.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, delOut.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if envelope.Data.ID != budgetID {
		t.Fatalf("deleted ID = %q, want %q", envelope.Data.ID, budgetID)
	}
	if envelope.Meta.Command != "budgets.delete" {
		t.Fatalf("command = %q, want budgets.delete", envelope.Meta.Command)
	}

	// Verify it's gone.
	var listOut, listErr bytes.Buffer
	exitCode = Run(context.Background(), []string{"--config", configPath, "budgets", "list", "--json"}, nil, &listOut, &listErr)
	if exitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", exitCode, listErr.String())
	}
	if strings.Contains(listOut.String(), "Disposable") {
		t.Fatalf("deleted budget still appears in list: %s", listOut.String())
	}
}

func TestBudgetCategoriesCreateDryRunJSONReturnsPlan(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "budgets", "categories", "create",
		"--budget-id", "fake-budget-id",
		"--name", "Groceries",
		"--limit", "50000",
		"--dry-run",
		"--json",
	}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			DryRun         bool `json:"dry_run"`
			BudgetCategory struct {
				Name     string `json:"name"`
				BudgetID string `json:"budget_id"`
				Limit    string `json:"limit"`
				Currency string `json:"currency"`
			} `json:"budget_category"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if !envelope.Data.DryRun {
		t.Fatal("dry_run = false, want true")
	}
	if envelope.Data.BudgetCategory.Name != "Groceries" {
		t.Fatalf("name = %q, want Groceries", envelope.Data.BudgetCategory.Name)
	}
	if envelope.Data.BudgetCategory.BudgetID != "fake-budget-id" {
		t.Fatalf("budget_id = %q, want fake-budget-id", envelope.Data.BudgetCategory.BudgetID)
	}
	if envelope.Meta.Command != "budgets.categories.create" {
		t.Fatalf("command = %q, want budgets.categories.create", envelope.Meta.Command)
	}
}

func TestBudgetCategoriesCreateDryRunHumanOutputsPlan(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "budgets", "categories", "create",
		"--budget-id", "fake-budget-id",
		"--name", "Dining",
		"--limit", "30000",
		"--dry-run",
	}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Would create budget category") {
		t.Fatalf("stdout missing dry-run message:\n%s", out)
	}
	if !strings.Contains(out, "Dining") {
		t.Fatalf("stdout missing category name:\n%s", out)
	}
}

func TestBudgetCategoriesDeleteJSONRemovesCategory(t *testing.T) {
	configPath := writeTestConfig(t, "")

	// Create a budget first.
	var budgetOut, budgetErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "create",
		"--name", "Monthly Budget",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--confirm",
		"--json",
	}, nil, &budgetOut, &budgetErr)
	if exitCode != 0 {
		t.Fatalf("budget create exit code = %d, stderr = %s", exitCode, budgetErr.String())
	}
	var budgetEnv struct {
		Data struct {
			Budget struct {
				ID string `json:"id"`
			} `json:"budget"`
		} `json:"data"`
	}
	if err := json.Unmarshal(budgetOut.Bytes(), &budgetEnv); err != nil {
		t.Fatalf("budget create output is not JSON: %v", err)
	}
	budgetID := budgetEnv.Data.Budget.ID

	// Create a budget category.
	var catOut, catErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "categories", "create",
		"--budget-id", budgetID,
		"--name", "Groceries",
		"--limit", "50000",
		"--confirm",
		"--json",
	}, nil, &catOut, &catErr)
	if exitCode != 0 {
		t.Fatalf("category create exit code = %d, stderr = %s", exitCode, catErr.String())
	}
	var catEnv struct {
		Data struct {
			BudgetCategory struct {
				ID string `json:"id"`
			} `json:"budget_category"`
		} `json:"data"`
	}
	if err := json.Unmarshal(catOut.Bytes(), &catEnv); err != nil {
		t.Fatalf("category create output is not JSON: %v", err)
	}
	categoryID := catEnv.Data.BudgetCategory.ID
	if categoryID == "" {
		t.Fatal("created category has empty ID")
	}

	// Delete the budget category.
	var delOut, delErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"budgets", "categories", "delete", categoryID,
		"--json",
	}, nil, &delOut, &delErr)
	if exitCode != 0 {
		t.Fatalf("delete exit code = %d, stderr = %s", exitCode, delErr.String())
	}

	var delEnvelope struct {
		OK   bool `json:"ok"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(delOut.Bytes(), &delEnvelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, delOut.String())
	}
	if !delEnvelope.OK {
		t.Fatal("ok = false")
	}
	if delEnvelope.Data.ID != categoryID {
		t.Fatalf("deleted ID = %q, want %q", delEnvelope.Data.ID, categoryID)
	}
	if delEnvelope.Meta.Command != "budgets.categories.delete" {
		t.Fatalf("command = %q, want budgets.categories.delete", delEnvelope.Meta.Command)
	}
}

func TestBudgetCreateJSONWithoutConfirmationReturnsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "budgets", "create",
		"--name", "Test",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--json",
	}, nil, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %s, stdout = %s", exitCode, stderr.String(), stdout.String())
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if envelope.OK {
		t.Fatal("ok = true, want false")
	}
	if len(envelope.Errors) != 1 || envelope.Errors[0].Code != "CONFIRMATION_REQUIRED" {
		t.Fatalf("errors = %#v", envelope.Errors)
	}
	if envelope.Meta.Command != "budgets.create" {
		t.Fatalf("command = %q, want budgets.create", envelope.Meta.Command)
	}
}

func TestBudgetCreateMissingNameReturnsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "budgets", "create",
		"--period", "monthly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--dry-run",
	}, nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--name") {
		t.Fatalf("stderr missing --name hint: %s", stderr.String())
	}
}

func TestBudgetCreateInvalidPeriodReturnsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "budgets", "create",
		"--name", "Test",
		"--period", "weekly",
		"--start-date", "2026-01-01",
		"--end-date", "2026-12-31",
		"--dry-run",
	}, nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--period") {
		t.Fatalf("stderr missing --period hint: %s", stderr.String())
	}
}

func TestBudgetCategoriesCreateJSONWithoutConfirmationReturnsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "budgets", "categories", "create",
		"--budget-id", "fake-id",
		"--name", "Groceries",
		"--limit", "50000",
		"--json",
	}, nil, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %s, stdout = %s", exitCode, stderr.String(), stdout.String())
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if envelope.OK {
		t.Fatal("ok = true, want false")
	}
	if len(envelope.Errors) != 1 || envelope.Errors[0].Code != "CONFIRMATION_REQUIRED" {
		t.Fatalf("errors = %#v", envelope.Errors)
	}
}

func TestBudgetCategoriesCreateMissingFieldsReturnsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "budgets", "categories", "create",
		"--name", "Groceries",
		"--dry-run",
	}, nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--budget-id") {
		t.Fatalf("stderr missing --budget-id hint: %s", stderr.String())
	}
}
