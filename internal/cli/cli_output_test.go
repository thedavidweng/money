package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/syncer"
)

func TestWriteSyncHumanNoItemsWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	writeSyncHuman(&buf, syncer.Result{}, false)
	if buf.Len() != 0 {
		t.Fatalf("output = %q, want empty for no items and no warnings", buf.String())
	}
}

func TestWriteSyncHumanWarnings(t *testing.T) {
	var buf bytes.Buffer
	result := syncer.Result{
		Warnings: []syncer.Warning{
			{Code: "NO_LINKED_PROVIDER_ITEMS", Message: "No linked Provider Items found."},
		},
	}
	writeSyncHuman(&buf, result, false)
	out := buf.String()
	if !strings.Contains(out, "warning") {
		t.Fatalf("output missing 'warning':\n%s", out)
	}
	if !strings.Contains(out, "NO_LINKED_PROVIDER_ITEMS") {
		t.Fatalf("output missing warning code:\n%s", out)
	}
	if !strings.Contains(out, "No linked Provider Items found.") {
		t.Fatalf("output missing warning message:\n%s", out)
	}
}

func TestWriteSyncHumanOkAndErrorCounts(t *testing.T) {
	var buf bytes.Buffer
	result := syncer.Result{
		Items: []syncer.ItemResult{
			{Provider: "plaid", ProviderItemID: "item_1", Status: "ok"},
			{Provider: "plaid", ProviderItemID: "item_2", Status: "ok"},
			{Provider: "bridge", ProviderItemID: "item_3", Status: "error"},
		},
	}
	writeSyncHuman(&buf, result, false)
	out := buf.String()
	if !strings.Contains(out, "ok=2") {
		t.Fatalf("output missing ok=2:\n%s", out)
	}
	if !strings.Contains(out, "errors=1") {
		t.Fatalf("output missing errors=1:\n%s", out)
	}
	if !strings.Contains(out, "synced") {
		t.Fatalf("output missing 'synced':\n%s", out)
	}
}

func TestWriteSyncHumanVerboseShowsPerItemDetails(t *testing.T) {
	var buf bytes.Buffer
	result := syncer.Result{
		Items: []syncer.ItemResult{
			{Provider: "plaid", ProviderItemID: "item_1", Status: "ok", AccountsSeen: 3, TransactionsAdded: 5, TransactionsModified: 2, TransactionsRemoved: 0},
			{Provider: "bridge", ProviderItemID: "item_2", Status: "error", ErrorCode: "PROVIDER_NOT_REGISTERED"},
		},
	}
	writeSyncHuman(&buf, result, true)
	out := buf.String()
	if !strings.Contains(out, "plaid") {
		t.Fatalf("output missing provider name:\n%s", out)
	}
	if !strings.Contains(out, "item_1") {
		t.Fatalf("output missing provider item ID:\n%s", out)
	}
	if !strings.Contains(out, "accounts=3") {
		t.Fatalf("output missing accounts count:\n%s", out)
	}
	if !strings.Contains(out, "added=5") {
		t.Fatalf("output missing added count:\n%s", out)
	}
	if !strings.Contains(out, "modified=2") {
		t.Fatalf("output missing modified count:\n%s", out)
	}
	// Verbose mode does not emit the summary line.
	if strings.Contains(out, "synced") {
		t.Fatalf("verbose mode should not emit summary line:\n%s", out)
	}
}

func TestWriteSyncHumanWarningsAndItemsTogether(t *testing.T) {
	var buf bytes.Buffer
	result := syncer.Result{
		Warnings: []syncer.Warning{
			{Code: "STALE_CURSOR", Message: "Cursor may be stale."},
		},
		Items: []syncer.ItemResult{
			{Provider: "plaid", ProviderItemID: "item_1", Status: "ok"},
		},
	}
	writeSyncHuman(&buf, result, false)
	out := buf.String()
	if !strings.Contains(out, "STALE_CURSOR") {
		t.Fatalf("output missing warning code:\n%s", out)
	}
	if !strings.Contains(out, "ok=1") {
		t.Fatalf("output missing ok=1:\n%s", out)
	}
}

func TestWriteSyncJSONSuccessReturnsEnvelope(t *testing.T) {
	var buf bytes.Buffer
	result := syncer.Result{
		Items: []syncer.ItemResult{
			{Provider: "plaid", ProviderItemID: "item_1", Status: "ok", TransactionsAdded: 3},
		},
		Warnings: []syncer.Warning{},
	}
	err := writeSyncJSON(&buf, result, nil)
	if err != nil {
		t.Fatalf("writeSyncJSON returned error: %v", err)
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Items []struct {
				Provider          string `json:"provider"`
				Status            string `json:"status"`
				TransactionsAdded int    `json:"transactions_added"`
			} `json:"items"`
			Warnings []any `json:"warnings"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, buf.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if envelope.Meta.Command != "sync" {
		t.Fatalf("command = %q, want sync", envelope.Meta.Command)
	}
	if len(envelope.Data.Items) != 1 {
		t.Fatalf("items length = %d, want 1", len(envelope.Data.Items))
	}
	if envelope.Data.Items[0].TransactionsAdded != 3 {
		t.Fatalf("transactions_added = %d, want 3", envelope.Data.Items[0].TransactionsAdded)
	}
}

func TestWriteSyncJSONPartialFailureReturnsErrorEnvelopeWithExitCode(t *testing.T) {
	var buf bytes.Buffer
	result := syncer.Result{
		Items: []syncer.ItemResult{
			{Provider: "plaid", ProviderItemID: "item_1", Status: "ok"},
			{Provider: "bridge", ProviderItemID: "item_2", Status: "error", ErrorCode: "PROVIDER_NOT_REGISTERED", ErrorMessage: "provider \"bridge\" is not registered"},
		},
	}
	partialErr := syncer.PartialFailure{Result: result}
	err := writeSyncJSON(&buf, result, partialErr)

	var exitErr cliExit
	if !errors.As(err, &exitErr) {
		t.Fatalf("error is not cliExit: %v", err)
	}
	if exitErr.exitCode != 6 {
		t.Fatalf("exit code = %d, want 6", exitErr.exitCode)
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Items []struct {
				Status string `json:"status"`
			} `json:"items"`
		} `json:"data"`
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, buf.String())
	}
	if envelope.OK {
		t.Fatal("ok = true, want false")
	}
	if len(envelope.Errors) != 1 || envelope.Errors[0].Code != "SYNC_PARTIAL_FAILURE" {
		t.Fatalf("errors = %#v", envelope.Errors)
	}
	if len(envelope.Data.Items) != 2 {
		t.Fatalf("data items length = %d, want 2 (partial failure includes result data)", len(envelope.Data.Items))
	}
	if envelope.Meta.Command != "sync" {
		t.Fatalf("command = %q, want sync", envelope.Meta.Command)
	}
}

func TestWriteSyncJSONGenericErrorReturnsRawError(t *testing.T) {
	var buf bytes.Buffer
	genericErr := fmt.Errorf("database connection failed")
	err := writeSyncJSON(&buf, syncer.Result{}, genericErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "database connection failed") {
		t.Fatalf("error = %q, want to contain 'database connection failed'", err.Error())
	}
	// Generic errors are NOT written as JSON; they propagate up.
	if buf.Len() != 0 {
		t.Fatalf("stdout should be empty for generic errors, got: %s", buf.String())
	}
}

func TestWriteSyncJSONSuccessWithEmptyResult(t *testing.T) {
	var buf bytes.Buffer
	err := writeSyncJSON(&buf, syncer.Result{}, nil)
	if err != nil {
		t.Fatalf("writeSyncJSON returned error: %v", err)
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Items    []any `json:"items"`
			Warnings []any `json:"warnings"`
		} `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, buf.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if len(envelope.Data.Items) != 0 {
		t.Fatalf("items length = %d, want 0", len(envelope.Data.Items))
	}
}

func TestColorAmountNonTerminalReturnsPlainAmount(t *testing.T) {
	// bytes.Buffer is not a terminal, so supportsColor returns false.
	var buf bytes.Buffer

	tests := []struct {
		input string
		want  string
	}{
		{"123.45", "123.45"},
		{"-50.00", "-50.00"},
		{"0", "0"},
		{"0.00", "0.00"},
		{"-0.00", "-0.00"},
		{"", ""},
		{"-", "-"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := colorAmount(&buf, tt.input)
			if got != tt.want {
				t.Fatalf("colorAmount(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestColorAmountFloatNonTerminalReturnsPlainAmount(t *testing.T) {
	var buf bytes.Buffer

	tests := []struct {
		input float64
		want  string
	}{
		{123.45, "123.45"},
		{-50.00, "-50.00"},
		{0.00, "0.00"},
		{-0.01, "-0.01"},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%.2f", tt.input)
		t.Run(name, func(t *testing.T) {
			got := colorAmountFloat(&buf, tt.input)
			if got != tt.want {
				t.Fatalf("colorAmountFloat(%f) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCommandNameFormatsCobraPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"money accounts list", ".accounts list"},
		{"money budgets create", ".budgets create"},
		{"money rules apply", ".rules apply"},
		{"money sync", ".sync"},
		{"money", "money"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			parts := strings.Split(tt.path, " ")
			if len(parts) == 1 {
				cmd := &cobra.Command{Use: parts[0]}
				got := commandName(cmd)
				if got != tt.want {
					t.Fatalf("commandName(%q) = %q, want %q", tt.path, got, tt.want)
				}
				return
			}
			// Build parent chain.
			root := &cobra.Command{Use: parts[0]}
			parent := root
			for i := 1; i < len(parts)-1; i++ {
				child := &cobra.Command{Use: parts[i]}
				parent.AddCommand(child)
				parent = child
			}
			leaf := &cobra.Command{Use: parts[len(parts)-1]}
			parent.AddCommand(leaf)
			got := commandName(leaf)
			if got != tt.want {
				t.Fatalf("commandName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestCommandNameNilCommandReturnsUnknown(t *testing.T) {
	got := commandName(nil)
	if got != "unknown" {
		t.Fatalf("commandName(nil) = %q, want unknown", got)
	}
}

func TestWriteSyncHumanAllErrorItems(t *testing.T) {
	var buf bytes.Buffer
	result := syncer.Result{
		Items: []syncer.ItemResult{
			{Provider: "plaid", ProviderItemID: "item_1", Status: "error", ErrorCode: "TIMEOUT"},
			{Provider: "bridge", ProviderItemID: "item_2", Status: "error", ErrorCode: "AUTH_EXPIRED"},
		},
	}
	writeSyncHuman(&buf, result, false)
	out := buf.String()
	if !strings.Contains(out, "ok=0") {
		t.Fatalf("output missing ok=0:\n%s", out)
	}
	if !strings.Contains(out, "errors=2") {
		t.Fatalf("output missing errors=2:\n%s", out)
	}
}

func TestWriteSyncHumanVerboseAllErrorItems(t *testing.T) {
	var buf bytes.Buffer
	result := syncer.Result{
		Items: []syncer.ItemResult{
			{Provider: "plaid", ProviderItemID: "item_1", Status: "error", ErrorCode: "TIMEOUT"},
			{Provider: "bridge", ProviderItemID: "item_2", Status: "error", ErrorCode: "AUTH_EXPIRED"},
		},
	}
	writeSyncHuman(&buf, result, true)
	out := buf.String()
	if !strings.Contains(out, "error") {
		t.Fatalf("verbose output missing 'error' status:\n%s", out)
	}
	// Should not have summary line in verbose mode.
	if strings.Contains(out, "synced") {
		t.Fatalf("verbose mode should not emit summary line:\n%s", out)
	}
}

func TestSyncJSONWithEmptyStoreReturnsWarningNoLinkedItems(t *testing.T) {
	configPath := writeTestConfig(t, "")

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"--config", configPath, "sync", "--json"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", exitCode, stderr.String(), stdout.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Items    []any `json:"items"`
			Warnings []struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"warnings"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatalf("ok = false, errors = %#v", envelope)
	}
	if envelope.Meta.Command != "sync" {
		t.Fatalf("command = %q, want sync", envelope.Meta.Command)
	}
	if len(envelope.Data.Warnings) == 0 {
		t.Fatal("warnings is empty, want at least one warning about no linked provider items")
	}
	found := false
	for _, w := range envelope.Data.Warnings {
		if w.Code == "NO_LINKED_PROVIDER_ITEMS" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("warnings = %#v, want NO_LINKED_PROVIDER_ITEMS", envelope.Data.Warnings)
	}
}

func TestSyncHumanWithEmptyStoreOutputsWarning(t *testing.T) {
	configPath := writeTestConfig(t, "")

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"--config", configPath, "sync"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "NO_LINKED_PROVIDER_ITEMS") {
		t.Fatalf("stdout missing warning code:\n%s", out)
	}
}

func TestSyncVerboseHumanWithEmptyStoreOutputsWarning(t *testing.T) {
	configPath := writeTestConfig(t, "")

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"--config", configPath, "sync", "--verbose"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "NO_LINKED_PROVIDER_ITEMS") {
		t.Fatalf("stdout missing warning code:\n%s", out)
	}
}

func TestManualPlanJSONHasCorrectCommand(t *testing.T) {
	var buf bytes.Buffer
	state := &runtimeState{demo: true, json: true}
	plan := manualAccountPlan{
		AccountName:       "Savings",
		SignedBalance:     "1000.00",
		FinancialPosition: "asset",
		WillWrite:         true,
	}
	err := writeManualPlan(&buf, state, plan)
	if err != nil {
		t.Fatalf("writeManualPlan returned error: %v", err)
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Plan struct {
				AccountName string `json:"account_name"`
				WillWrite   bool   `json:"will_write"`
			} `json:"plan"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
			Demo    bool   `json:"demo"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, buf.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if envelope.Data.Plan.AccountName != "Savings" {
		t.Fatalf("account_name = %q, want Savings", envelope.Data.Plan.AccountName)
	}
	if envelope.Meta.Command != "accounts.create_manual" {
		t.Fatalf("command = %q, want accounts.create_manual", envelope.Meta.Command)
	}
	if !envelope.Meta.Demo {
		t.Fatal("meta.demo = false, want true")
	}
}
