package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestDemoRulesListJSONReturnsEnvelopeWithEmptyRules(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "rules", "list", "--json"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Rules []json.RawMessage `json:"rules"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
			Demo    bool   `json:"demo"`
		} `json:"meta"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON envelope: %v\n%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatalf("ok = false, error = %#v", envelope.Error)
	}
	if envelope.Meta.Command != "rules.list" {
		t.Fatalf("command = %q, want rules.list", envelope.Meta.Command)
	}
	if !envelope.Meta.Demo {
		t.Fatal("meta.demo = false, want true")
	}
	if len(envelope.Data.Rules) != 0 {
		t.Fatalf("rules length = %d, want 0 for empty demo store", len(envelope.Data.Rules))
	}
}

func TestDemoRulesListHumanOutputsTable(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "rules", "list"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable table:\n%s", stdout.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "IF") || !strings.Contains(out, "THEN") {
		t.Fatalf("stdout missing expected table headers:\n%s", out)
	}
}

func TestRuleCreateDryRunJSONReturnsPlanWithoutPersisting(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "rules", "create",
		"--name", "Mark Uber",
		"--condition-field", "merchant_name",
		"--condition-op", "contains",
		"--condition-value", "uber",
		"--action-type", "set_category",
		"--action-value", "transport",
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
			Rule   struct {
				Name           string `json:"name"`
				ConditionField string `json:"condition_field"`
				ConditionOp    string `json:"condition_op"`
				ConditionValue string `json:"condition_value"`
				ActionType     string `json:"action_type"`
				ActionValue    string `json:"action_value"`
				Priority       int    `json:"priority"`
				Enabled        bool   `json:"enabled"`
			} `json:"rule"`
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
	if envelope.Data.Rule.Name != "Mark Uber" {
		t.Fatalf("rule name = %q, want Mark Uber", envelope.Data.Rule.Name)
	}
	if envelope.Data.Rule.ConditionField != "merchant_name" {
		t.Fatalf("condition_field = %q, want merchant_name", envelope.Data.Rule.ConditionField)
	}
	if envelope.Data.Rule.ConditionOp != "contains" {
		t.Fatalf("condition_op = %q, want contains", envelope.Data.Rule.ConditionOp)
	}
	if envelope.Data.Rule.ActionType != "set_category" {
		t.Fatalf("action_type = %q, want set_category", envelope.Data.Rule.ActionType)
	}
	if !envelope.Data.Rule.Enabled {
		t.Fatal("enabled = false, want true")
	}
	if envelope.Meta.Command != "rules.create" {
		t.Fatalf("command = %q, want rules.create", envelope.Meta.Command)
	}

	// Verify dry-run did not persist.
	var listOut, listErr bytes.Buffer
	exitCode = Run(context.Background(), []string{"demo", "rules", "list", "--json"}, nil, &listOut, &listErr)
	if exitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", exitCode, listErr.String())
	}
	if strings.Contains(listOut.String(), "Mark Uber") {
		t.Fatalf("dry-run rule persisted in demo store: %s", listOut.String())
	}
}

func TestRuleCreateDryRunHumanOutputsPlan(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "rules", "create",
		"--name", "Mark Uber",
		"--condition-field", "merchant_name",
		"--condition-op", "contains",
		"--condition-value", "uber",
		"--action-type", "set_category",
		"--action-value", "transport",
		"--dry-run",
	}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Would create rule") {
		t.Fatalf("stdout missing dry-run message:\n%s", out)
	}
	if !strings.Contains(out, "Mark Uber") {
		t.Fatalf("stdout missing rule name:\n%s", out)
	}
	if !strings.Contains(out, "uber") {
		t.Fatalf("stdout missing condition value:\n%s", out)
	}
}

func TestRuleCreateConfirmJSONPersistsAndReturnsCreatedRule(t *testing.T) {
	configPath := writeTestConfig(t)

	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"rules", "create",
		"--name", "Groceries Rule",
		"--condition-field", "merchant_name",
		"--condition-op", "contains",
		"--condition-value", "whole foods",
		"--action-type", "set_category",
		"--action-value", "groceries",
		"--confirm",
		"--json",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s, stdout = %s", exitCode, createErr.String(), createOut.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Rule struct {
				ID             string `json:"id"`
				Name           string `json:"name"`
				ConditionField string `json:"condition_field"`
				ConditionOp    string `json:"condition_op"`
				ConditionValue string `json:"condition_value"`
				ActionType     string `json:"action_type"`
				ActionValue    string `json:"action_value"`
				Enabled        bool   `json:"enabled"`
			} `json:"rule"`
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
	if envelope.Data.Rule.ID == "" {
		t.Fatal("created rule ID is empty")
	}
	if envelope.Data.Rule.Name != "Groceries Rule" {
		t.Fatalf("rule name = %q, want Groceries Rule", envelope.Data.Rule.Name)
	}
	if !envelope.Data.Rule.Enabled {
		t.Fatal("enabled = false, want true")
	}
	if envelope.Meta.Command != "rules.create" {
		t.Fatalf("command = %q, want rules.create", envelope.Meta.Command)
	}

	// Verify it persists by listing.
	var listOut, listErr bytes.Buffer
	exitCode = Run(context.Background(), []string{"--config", configPath, "rules", "list", "--json"}, nil, &listOut, &listErr)
	if exitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", exitCode, listErr.String())
	}
	if !strings.Contains(listOut.String(), "Groceries Rule") {
		t.Fatalf("created rule missing from list: %s", listOut.String())
	}
}

func TestRuleCreateConfirmHumanOutputsCreatedRule(t *testing.T) {
	configPath := writeTestConfig(t)

	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"rules", "create",
		"--name", "Categorize Uber",
		"--condition-field", "merchant_name",
		"--condition-op", "contains",
		"--condition-value", "uber",
		"--action-type", "set_category",
		"--action-value", "transport",
		"--confirm",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", exitCode, createErr.String())
	}
	if json.Valid(createOut.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable:\n%s", createOut.String())
	}
	out := createOut.String()
	if !strings.Contains(out, "Created rule") {
		t.Fatalf("stdout missing 'Created rule':\n%s", out)
	}
	if !strings.Contains(out, "Categorize Uber") {
		t.Fatalf("stdout missing rule name:\n%s", out)
	}
}

func TestRuleDeleteHumanOutputsDeletedRule(t *testing.T) {
	configPath := writeTestConfig(t)

	// Create a rule.
	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"rules", "create",
		"--name", "ToDelete",
		"--condition-field", "name",
		"--condition-op", "equals",
		"--condition-value", "Test",
		"--action-type", "set_note",
		"--action-value", "test",
		"--confirm",
		"--json",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", exitCode, createErr.String())
	}
	var created struct {
		Data struct {
			Rule struct {
				ID string `json:"id"`
			} `json:"rule"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createOut.Bytes(), &created); err != nil {
		t.Fatalf("create output is not JSON: %v", err)
	}

	// Delete in human mode.
	var delOut, delErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"rules", "delete", created.Data.Rule.ID,
	}, nil, &delOut, &delErr)
	if exitCode != 0 {
		t.Fatalf("delete exit code = %d, stderr = %s", exitCode, delErr.String())
	}
	if json.Valid(delOut.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable:\n%s", delOut.String())
	}
	out := delOut.String()
	if !strings.Contains(out, "Deleted rule") {
		t.Fatalf("stdout missing 'Deleted rule':\n%s", out)
	}
	if !strings.Contains(out, created.Data.Rule.ID) {
		t.Fatalf("stdout missing rule ID:\n%s", out)
	}
}

func TestRulesListHumanWithRealRulesShowsTable(t *testing.T) {
	configPath := writeTestConfig(t)

	// Create a rule.
	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"rules", "create",
		"--name", "Coffee Shops",
		"--condition-field", "merchant_name",
		"--condition-op", "contains",
		"--condition-value", "coffee",
		"--action-type", "set_category",
		"--action-value", "coffee_shops",
		"--confirm",
		"--json",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", exitCode, createErr.String())
	}

	// List in human mode.
	var listOut, listErr bytes.Buffer
	exitCode = Run(context.Background(), []string{"--config", configPath, "rules", "list"}, nil, &listOut, &listErr)
	if exitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", exitCode, listErr.String())
	}
	if json.Valid(listOut.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable:\n%s", listOut.String())
	}
	out := listOut.String()
	if !strings.Contains(out, "Coffee Shops") {
		t.Fatalf("stdout missing rule name:\n%s", out)
	}
	if !strings.Contains(out, "contains") {
		t.Fatalf("stdout missing condition operator:\n%s", out)
	}
	if !strings.Contains(out, "set_category") {
		t.Fatalf("stdout missing action type:\n%s", out)
	}
}

func TestRuleDeleteJSONRemovesRule(t *testing.T) {
	configPath := writeTestConfig(t)

	// Create a rule.
	var createOut, createErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"rules", "create",
		"--name", "Disposable Rule",
		"--condition-field", "name",
		"--condition-op", "equals",
		"--condition-value", "Starbucks",
		"--action-type", "set_note",
		"--action-value", "coffee",
		"--confirm",
		"--json",
	}, nil, &createOut, &createErr)
	if exitCode != 0 {
		t.Fatalf("create exit code = %d, stderr = %s", exitCode, createErr.String())
	}
	var created struct {
		Data struct {
			Rule struct {
				ID string `json:"id"`
			} `json:"rule"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createOut.Bytes(), &created); err != nil {
		t.Fatalf("create output is not JSON: %v", err)
	}
	ruleID := created.Data.Rule.ID

	// Delete the rule.
	var delOut, delErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"rules", "delete", ruleID,
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
	if envelope.Data.ID != ruleID {
		t.Fatalf("deleted ID = %q, want %q", envelope.Data.ID, ruleID)
	}
	if envelope.Meta.Command != "rules.delete" {
		t.Fatalf("command = %q, want rules.delete", envelope.Meta.Command)
	}

	// Verify it's gone.
	var listOut, listErr bytes.Buffer
	exitCode = Run(context.Background(), []string{"--config", configPath, "rules", "list", "--json"}, nil, &listOut, &listErr)
	if exitCode != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", exitCode, listErr.String())
	}
	if strings.Contains(listOut.String(), "Disposable Rule") {
		t.Fatalf("deleted rule still appears in list: %s", listOut.String())
	}
}

func TestRuleApplyJSONReturnsResult(t *testing.T) {
	configPath := writeTestConfig(t)

	// Create a rule.
	var ruleOut, ruleErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"rules", "create",
		"--name", "Categorize Coffee",
		"--condition-field", "merchant_name",
		"--condition-op", "contains",
		"--condition-value", "coffee",
		"--action-type", "set_category",
		"--action-value", "coffee_shops",
		"--confirm",
		"--json",
	}, nil, &ruleOut, &ruleErr)
	if exitCode != 0 {
		t.Fatalf("rule create exit code = %d, stderr = %s", exitCode, ruleErr.String())
	}

	// Apply rules (no transactions exist yet, so 0 updated).
	var applyOut, applyErr bytes.Buffer
	exitCode = Run(context.Background(), []string{
		"--config", configPath,
		"rules", "apply",
		"--json",
	}, nil, &applyOut, &applyErr)
	if exitCode != 0 {
		t.Fatalf("apply exit code = %d, stderr = %s", exitCode, applyErr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Result struct {
				TransactionsUpdated int `json:"transactions_updated"`
			} `json:"result"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(applyOut.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, applyOut.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if envelope.Data.Result.TransactionsUpdated != 0 {
		t.Fatalf("transactions_updated = %d, want 0 (no transactions in store)", envelope.Data.Result.TransactionsUpdated)
	}
	if envelope.Meta.Command != "rules.apply" {
		t.Fatalf("command = %q, want rules.apply", envelope.Meta.Command)
	}
}

func TestRuleApplyHumanOutputsUpdatedCount(t *testing.T) {
	configPath := writeTestConfig(t)

	// Apply rules in human mode.
	var applyOut, applyErr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"rules", "apply",
	}, nil, &applyOut, &applyErr)
	if exitCode != 0 {
		t.Fatalf("apply exit code = %d, stderr = %s", exitCode, applyErr.String())
	}
	out := applyOut.String()
	if !strings.Contains(out, "Updated") || !strings.Contains(out, "transactions") {
		t.Fatalf("stdout missing updated message:\n%s", out)
	}
}

func TestRuleCreateJSONWithoutConfirmationReturnsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "rules", "create",
		"--name", "Test Rule",
		"--condition-field", "merchant_name",
		"--condition-op", "contains",
		"--condition-value", "test",
		"--action-type", "set_category",
		"--action-value", "test_cat",
		"--json",
	}, nil, &stdout, &stderr)
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want 7; stderr = %s, stdout = %s", exitCode, stderr.String(), stdout.String())
	}

	var envelope struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
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
	if envelope.Error == nil || envelope.Error.Code != "CONFIRMATION_REQUIRED" {
		t.Fatalf("error = %#v", envelope.Error)
	}
	if envelope.Meta.Command != "rules.create" {
		t.Fatalf("command = %q, want rules.create", envelope.Meta.Command)
	}
}

func TestRuleCreateMissingFieldsReturnsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "rules", "create",
		"--name", "Test Rule",
		"--dry-run",
	}, nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--condition-field") {
		t.Fatalf("stderr missing required field hint: %s", stderr.String())
	}
}

func TestRuleCreateWithPriorityDryRunIncludesPriority(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"demo", "rules", "create",
		"--name", "High Priority Rule",
		"--condition-field", "name",
		"--condition-op", "equals",
		"--condition-value", "Amazon",
		"--action-type", "set_category",
		"--action-value", "shopping",
		"--priority", "10",
		"--dry-run",
		"--json",
	}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Rule struct {
				Priority int `json:"priority"`
			} `json:"rule"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if envelope.Data.Rule.Priority != 10 {
		t.Fatalf("priority = %d, want 10", envelope.Data.Rule.Priority)
	}
}
