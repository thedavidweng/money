package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestDemoInvestmentsHoldingsJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "investments", "holdings", "--json"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Holdings []struct {
				AccountID  string  `json:"account_id"`
				SecurityID string  `json:"security_id"`
				Quantity   float64 `json:"quantity"`
				Currency   string  `json:"currency"`
			} `json:"holdings"`
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
	if envelope.Meta.Command != "investments.holdings" {
		t.Fatalf("command = %q, want investments.holdings", envelope.Meta.Command)
	}
	if !envelope.Meta.Demo {
		t.Fatal("meta.demo = false, want true")
	}
	// Demo store has no investment holdings; the envelope is still valid.
	_ = envelope.Data.Holdings
}

func TestDemoInvestmentsHoldingsHumanMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "investments", "holdings"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable output:\n%s", stdout.String())
	}
	for _, want := range []string{"ACCOUNT", "SECURITY", "QUANTITY", "PRICE", "VALUE"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing header %q:\n%s", want, stdout.String())
		}
	}
}

func TestDemoInvestmentsSecuritiesJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "investments", "securities", "--json"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Securities []struct {
				SecurityID string `json:"security_id"`
				Name       string `json:"name"`
				Currency   string `json:"currency"`
			} `json:"securities"`
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
	if envelope.Meta.Command != "investments.securities" {
		t.Fatalf("command = %q, want investments.securities", envelope.Meta.Command)
	}
	if !envelope.Meta.Demo {
		t.Fatal("meta.demo = false, want true")
	}
	// Demo store has no investment securities; the envelope is still valid.
	_ = envelope.Data.Securities
}

func TestDemoInvestmentsSecuritiesHumanMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "investments", "securities"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable output:\n%s", stdout.String())
	}
	for _, want := range []string{"SECURITY ID", "NAME", "TICKER", "TYPE", "CLOSE PRICE"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing header %q:\n%s", want, stdout.String())
		}
	}
}

func TestDemoLiabilitiesListJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "liabilities", "list", "--json"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Liabilities []struct {
				AccountID      string  `json:"account_id"`
				Type           string  `json:"type"`
				Name           string  `json:"name"`
				CurrentBalance float64 `json:"current_balance"`
				Currency       string  `json:"currency"`
			} `json:"liabilities"`
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
	if envelope.Meta.Command != "liabilities.list" {
		t.Fatalf("command = %q, want liabilities.list", envelope.Meta.Command)
	}
	if !envelope.Meta.Demo {
		t.Fatal("meta.demo = false, want true")
	}
	// Demo store has no liabilities; the envelope is still valid.
	_ = envelope.Data.Liabilities
}

func TestDemoLiabilitiesListHumanMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "liabilities", "list"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable output:\n%s", stdout.String())
	}
	for _, want := range []string{"ACCOUNT", "TYPE", "NAME", "BALANCE"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing header %q:\n%s", want, stdout.String())
		}
	}
}

func TestNetWorthJSON(t *testing.T) {
	configPath := writeTestConfig(t)

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"--config", configPath, "net-worth", "--json"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", exitCode, stderr.String(), stdout.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			NetWorth json.RawMessage `json:"net_worth"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON envelope: %v\n%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if envelope.Meta.Command != "net_worth" {
		t.Fatalf("command = %q, want net_worth", envelope.Meta.Command)
	}
	if !json.Valid(envelope.Data.NetWorth) {
		t.Fatalf("net_worth is not valid JSON: %s", string(envelope.Data.NetWorth))
	}
}

func TestNetWorthHumanMode(t *testing.T) {
	configPath := writeTestConfig(t)

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"--config", configPath, "net-worth"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable output:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Net worth") {
		t.Fatalf("stdout missing 'Net worth':\n%s", stdout.String())
	}
}

func TestCashflowJSON(t *testing.T) {
	configPath := writeTestConfig(t)

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"cashflow",
		"--from", "2024-01-01",
		"--to", "2024-12-31",
		"--json",
	}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", exitCode, stderr.String(), stdout.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Periods json.RawMessage `json:"periods"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON envelope: %v\n%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if envelope.Meta.Command != "cashflow" {
		t.Fatalf("command = %q, want cashflow", envelope.Meta.Command)
	}
}

func TestCashflowHumanMode(t *testing.T) {
	configPath := writeTestConfig(t)

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{
		"--config", configPath,
		"cashflow",
		"--from", "2024-01-01",
		"--to", "2024-12-31",
	}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable output:\n%s", stdout.String())
	}
	for _, want := range []string{"PERIOD", "INCOME", "EXPENSES", "NET"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing header %q:\n%s", want, stdout.String())
		}
	}
}

func TestDemoAccountsListJSONVerbose(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"demo", "accounts", "list", "--json", "--verbose"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Accounts []struct {
				ID               string  `json:"id"`
				DisplayName      string  `json:"display_name"`
				Type             string  `json:"type"`
				CurrentBalance   string  `json:"current_balance"`
				AvailableBalance *string `json:"available_balance"`
				AvailableCredit  *string `json:"available_credit"`
				Currency         string  `json:"currency"`
				UpdatedAt        string  `json:"updated_at"`
				Source           struct {
					Kind     string  `json:"kind"`
					Provider *string `json:"provider"`
				} `json:"source"`
			} `json:"accounts"`
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
	if envelope.Meta.Command != "accounts.list" {
		t.Fatalf("command = %q, want accounts.list", envelope.Meta.Command)
	}
	if !envelope.Meta.Demo {
		t.Fatal("meta.demo = false, want true")
	}
	if len(envelope.Data.Accounts) == 0 {
		t.Fatal("accounts is empty, want demo fixtures")
	}
	if envelope.Data.Accounts[0].ID == "" {
		t.Fatal("verbose account id is empty")
	}
	if envelope.Data.Accounts[0].UpdatedAt == "" {
		t.Fatal("verbose account updated_at is empty")
	}
}

func TestFeedbackJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"feedback", "--json"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON envelope: %v\n%s", err, stdout.String())
	}
	if !envelope.OK {
		t.Fatal("ok = false")
	}
	if envelope.Meta.Command != "feedback" {
		t.Fatalf("command = %q, want feedback", envelope.Meta.Command)
	}
	if envelope.Data.URL != "https://github.com/thedavidweng/money/issues" {
		t.Fatalf("url = %q, want GitHub issues URL", envelope.Data.URL)
	}
}

func TestFeedbackHumanMode(t *testing.T) {
	oldOpen := openBrowser
	t.Cleanup(func() { openBrowser = oldOpen })
	openBrowser = func(url string) error { return nil }

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"feedback"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout is JSON, want human-readable output:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "https://github.com/thedavidweng/money/issues") {
		t.Fatalf("stdout missing GitHub issues URL:\n%s", stdout.String())
	}
}

func TestWriteProviderAvailability(t *testing.T) {
	var buf bytes.Buffer
	rows := []providerAvailabilityRow{
		{Provider: "plaid", Status: "available", Code: "", Guidance: ""},
		{Provider: "bridge", Status: "unavailable", Code: "MISSING_CREDENTIALS", Guidance: "Configure providers.bridge credentials with env references."},
	}
	writeProviderAvailability(&buf, rows)

	output := buf.String()
	for _, want := range []string{
		"provider\tstatus\tcode\tguidance",
		"plaid\tavailable\t\t",
		"bridge\tunavailable\tMISSING_CREDENTIALS\tConfigure providers.bridge credentials with env references.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestWriteProviderAvailabilityEmpty(t *testing.T) {
	var buf bytes.Buffer
	writeProviderAvailability(&buf, nil)
	output := buf.String()
	if !strings.Contains(output, "provider\tstatus\tcode\tguidance") {
		t.Fatalf("output missing header:\n%s", output)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1 (header only):\n%s", len(lines), output)
	}
}

func TestSupportedProviderAvailabilityNoDiagnostics(t *testing.T) {
	rows := supportedProviderAvailability("plaid", nil)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].Provider != "plaid" {
		t.Fatalf("provider = %q, want plaid", rows[0].Provider)
	}
	if rows[0].Status != "available" {
		t.Fatalf("status = %q, want available", rows[0].Status)
	}
}
