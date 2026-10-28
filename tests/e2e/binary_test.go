package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// ─── Build cache ───

var (
	buildOnce sync.Once
	cachedBin string
	buildErr  error
)

func buildBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		projectRoot, err := findProjectRoot()
		if err != nil {
			buildErr = err
			return
		}
		dir, err := os.MkdirTemp("", "money-e2e-*")
		if err != nil {
			buildErr = err
			return
		}
		cachedBin = filepath.Join(dir, "money")
		if runtime.GOOS == "windows" {
			cachedBin += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", cachedBin, "./cmd/money")
		cmd.Dir = projectRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			buildErr = fmt.Errorf("build failed: %v\n%s", err, out)
			return
		}
	})
	if buildErr != nil {
		t.Fatalf("binary build failed: %v", buildErr)
	}
	return cachedBin
}

func findProjectRoot() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := pwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Development", "money"), nil
}

// ─── Execution helper ───

func run(t *testing.T, bin string, args ...string) (stdout string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	home := t.TempDir()
	cmd.Env = []string{
		"HOME=" + home,
		"USERPROFILE=" + home, // Windows uses USERPROFILE for home directory
		"PATH=/usr/bin:/bin:/usr/local/bin",
		"TERM=dumb",
		"NO_COLOR=1",
	}
	outBytes, err := cmd.CombinedOutput()
	stdout = string(outBytes)
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}
	return stdout, exitCode
}

func assertValidEnvelope(t *testing.T, stdout, wantCommand string) map[string]any {
	t.Helper()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("invalid JSON envelope: %v\noutput: %s", err, stdout)
	}
	if envelope["ok"] != true {
		t.Fatalf("envelope.ok = false: %+v", envelope)
	}
	if _, ok := envelope["data"]; !ok {
		t.Fatalf("envelope missing 'data': %+v", envelope)
	}
	if meta, ok := envelope["meta"].(map[string]any); ok {
		if got, ok := meta["command"].(string); ok && wantCommand != "" && got != wantCommand {
			t.Fatalf("command = %q, want %q", got, wantCommand)
		}
		if _, ok := meta["schema_version"]; !ok {
			t.Fatalf("meta missing 'schema_version': %+v", meta)
		}
	} else {
		t.Fatalf("envelope.meta is not an object: %+v", envelope)
	}
	return envelope
}

func requireZero(t *testing.T, code int, stdout string) {
	t.Helper()
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. output:\n%s", code, stdout)
	}
}

// ─── Individual command tests ───

func TestBinary_Doctor(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "doctor")
	if code != 1 {
		t.Fatalf("expected exit 1 for doctor without config, got %d", code)
	}
	if !strings.Contains(stdout, "config") {
		t.Fatalf("doctor output missing config reference: %q", stdout)
	}
}

func TestBinary_Doctor_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "doctor", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "doctor")
}

func TestBinary_Version(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "version")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "money") {
		t.Fatalf("version output missing 'money': %q", stdout)
	}
}

func TestBinary_Version_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "version", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "version")
}

// ─── Demo mode JSON schema tests ───

func TestBinary_Demo_Accounts_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "accounts", "list", "--json")
	requireZero(t, code, stdout)
	env := assertValidEnvelope(t, stdout, "accounts.list")
	meta, ok := env["meta"].(map[string]any)
	if !ok || meta["demo"] != true {
		t.Fatal("expected demo=true")
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data is not an object")
	}
	accounts, ok := data["accounts"].([]any)
	if !ok || len(accounts) < 1 {
		t.Fatal("expected at least 1 account")
	}
}

func TestBinary_Demo_Categories_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "categories", "list", "--json")
	requireZero(t, code, stdout)
	env := assertValidEnvelope(t, stdout, "categories.list")
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data is not an object")
	}
	categories, ok := data["categories"].([]any)
	if !ok || len(categories) < 1 {
		t.Fatal("expected at least 1 category")
	}
}

func TestBinary_Demo_Tags_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "tags", "list", "--json")
	requireZero(t, code, stdout)
	env := assertValidEnvelope(t, stdout, "tags.list")
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data is not an object")
	}
	tags, ok := data["tags"].([]any)
	if !ok || len(tags) < 1 {
		t.Fatal("expected at least 1 tag")
	}
}

func TestBinary_Demo_Transactions_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "transactions", "list", "--json")
	requireZero(t, code, stdout)
	env := assertValidEnvelope(t, stdout, "transactions.list")
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data is not an object")
	}
	transactions, ok := data["transactions"].([]any)
	if !ok || len(transactions) < 1 {
		t.Fatal("expected at least 1 transaction")
	}
}

func TestBinary_Demo_Transactions_Search_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "transactions", "search", "coffee", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "transactions.search")
}

func TestBinary_Demo_Recurring_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "recurring", "list", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "recurring.list")
}

func TestBinary_Demo_Investments_Holdings_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "investments", "holdings", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "investments.holdings")
}

func TestBinary_Demo_Investments_Securities_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "investments", "securities", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "investments.securities")
}

func TestBinary_Demo_Items_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "items", "list", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "items.list")
}

func TestBinary_Demo_Liabilities_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "liabilities", "list", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "liabilities.list")
}

func TestBinary_Demo_Budgets_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "budgets", "list", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "budgets.list")
}

func TestBinary_Demo_Rules_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "rules", "list", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "rules.list")
}

func TestBinary_Demo_NetWorth_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "net-worth", "--json")
	if code == 0 {
		assertValidEnvelope(t, stdout, "net_worth")
		return
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected JSON error envelope, got invalid JSON: %v\noutput: %s", err, stdout)
	}
	if envelope["ok"] != false {
		t.Fatalf("error envelope should have ok=false: %+v", envelope)
	}
}

func TestBinary_Demo_Cashflow_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "cashflow", "--json", "--from", "2026-01-01", "--to", "2026-12-31")
	if code == 0 {
		assertValidEnvelope(t, stdout, "cashflow")
		return
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected JSON error envelope, got invalid JSON: %v\noutput: %s", err, stdout)
	}
	if envelope["ok"] != false {
		t.Fatalf("error envelope should have ok=false: %+v", envelope)
	}
}

// ─── Edge cases ───

func TestBinary_UnknownCommand(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "nonexistent")
	if code == 0 {
		t.Fatalf("expected non-zero exit, got 0. output: %s", stdout)
	}
}

func TestBinary_EmptyArgs_ShowsHelp(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin)
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("no args should show help, got: %q", stdout)
	}
}

func TestBinary_AccountsList_WithoutConfig(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "accounts", "list", "--json")
	if code == 0 {
		t.Fatalf("expected non-zero exit for accounts list without config, got 0. output:\n%s", stdout)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("expected JSON error envelope, got invalid JSON: %v\noutput: %s", err, stdout)
	}
	if envelope["ok"] != false {
		t.Fatalf("error envelope should have ok=false: %+v", envelope)
	}
}

func TestBinary_ConfigFlag_Missing(t *testing.T) {
	bin := buildBinary(t)
	fakeConfig := filepath.Join(t.TempDir(), "fake-config.yaml")
	stdout, code := run(t, bin, "doctor", "--config", fakeConfig)
	if code != 1 {
		t.Fatalf("expected exit 1 for doctor with missing config, got %d", code)
	}
	if !strings.Contains(stdout, "config") {
		t.Fatalf("doctor output missing config reference: %q", stdout)
	}
}

func TestBinary_ProfileFlag(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "version", "--profile", "test")
	requireZero(t, code, stdout)
}

func TestBinary_GlobalFlags_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "version", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "version")
}

func TestBinary_Demo_Accounts_Plain(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "accounts", "list")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "NAME") {
		t.Fatalf("plain text accounts list missing NAME header: %q", stdout)
	}
}

func TestBinary_Demo_Transactions_Pagination(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "transactions", "list", "--json", "--limit", "2")
	requireZero(t, code, stdout)
	env := assertValidEnvelope(t, stdout, "transactions.list")
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data is not an object")
	}
	transactions, ok := data["transactions"].([]any)
	if !ok || len(transactions) > 2 {
		t.Fatalf("expected at most 2 transactions with --limit=2, got %d", len(transactions))
	}
}

func TestBinary_Demo_Transactions_Filter_NeedsReview(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "transactions", "list", "--json", "--needs-review", "true")
	requireZero(t, code, stdout)
	env := assertValidEnvelope(t, stdout, "transactions.list")
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data is not an object")
	}
	transactions, ok := data["transactions"].([]any)
	if !ok {
		t.Fatal("data.transactions is not an array")
	}
	for i, raw := range transactions {
		tx, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("transaction %d is not an object", i)
		}
		if needsReview, ok := tx["needs_review"].(bool); ok && !needsReview {
			t.Errorf("transaction %d has needs_review=false but filter was true", i)
		}
	}
}
