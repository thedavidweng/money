package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
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
	for dir := pwd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
	}
	return filepath.Join(os.Getenv("HOME"), "Development", "money"), nil
}

// ─── Execution helper ───

var (
	executedCmds = make(map[string]bool)
	mu           sync.Mutex
)

func recordCommand(args []string) {
	if len(args) == 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	var path string
	if len(args) >= 2 && !strings.HasPrefix(args[1], "-") {
		path = args[0] + " " + args[1]
	} else {
		path = args[0]
	}
	executedCmds[path] = true
}

func run(t *testing.T, bin string, args ...string) (stdout string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = []string{
		"HOME=" + t.TempDir(),
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
	recordCommand(args)
	return stdout, exitCode
}

func assertValidEnvelope(t *testing.T, stdout string, wantCommand string) map[string]any {
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

// ─── Command discovery from help output ───

// helpCmdPattern matches lines that start with 2 spaces, then a command name.
// Cobra groups commands under headers; we just look for the command lines.
// Example: "  demo         Run a command..." or "  accounts     "
var helpCmdPattern = regexp.MustCompile(`^  ([a-z][a-z0-9-]*)(?:$| +)`)

func discoverCommands(t *testing.T, bin string) []string {
	t.Helper()
	stdout, _ := run(t, bin, "--help")
	var cmds []string
	inFlags := false
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "Flags:") || strings.HasPrefix(line, "Global Flags:") {
			inFlags = true
		}
		if inFlags {
			continue
		}
		if m := helpCmdPattern.FindStringSubmatch(line); m != nil {
			cmd := m[1]
			if cmd == "help" || cmd == "completion" || cmd == "money" {
				continue
			}
			cmds = append(cmds, cmd)
		}
	}
	sort.Strings(cmds)
	return cmds
}

// ─── The golden list of expected commands ───
//
// AGENT INSTRUCTION: When you add a new command to money:
// 1. Add it to this list
// 2. Add a test below that exercises it
// 3. Run go test -v ./tests/e2e/... to verify

var requiredCommands = []string{
	"demo", "doctor", "setup",
	"accounts", "budgets", "cashflow", "categories",
	"import", "investments", "items", "liabilities",
	"net-worth", "recurring", "rules", "sync",
	"tags", "transactions", "tx",
	"link", "plaid", "providers",
	"feedback", "version",
}

// ─── Meta tests ───

func TestAllCommandsInHelp(t *testing.T) {
	bin := buildBinary(t)
	discovered := discoverCommands(t, bin)

	for _, required := range requiredCommands {
		found := false
		for _, d := range discovered {
			if d == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required command %q not found in help. Add the command to Cobra OR remove it from requiredCommands.", required)
		}
	}

	for _, d := range discovered {
		known := false
		for _, required := range requiredCommands {
			if d == required {
				known = true
				break
			}
		}
		if !known {
			t.Errorf("command %q appears in help but is NOT in requiredCommands. Add it to the list (and add a test below).", d)
		}
	}
}

// ─── Individual command tests ───

func TestBinary_Demo_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "demo", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "sandbox") {
		t.Error("demo help missing 'sandbox'")
	}
}

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

func TestBinary_Setup_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "setup", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "Usage:") {
		t.Fatal("setup help missing Usage:")
	}
}

func TestBinary_Accounts_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "accounts", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("accounts help missing 'list'")
	}
	if !strings.Contains(stdout, "create-manual") {
		t.Errorf("accounts help missing 'create-manual'")
	}
}

func TestBinary_Budgets_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "budgets", "--help")
	requireZero(t, code, stdout)
}

func TestBinary_Cashflow_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "cashflow", "--help")
	requireZero(t, code, stdout)
	for _, flag := range []string{"--from", "--to"} {
		if !strings.Contains(stdout, flag) {
			t.Errorf("cashflow help missing flag %q", flag)
		}
	}
}

func TestBinary_Categories_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "categories", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("categories help missing 'list'")
	}
}

func TestBinary_Import_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "import", "--help")
	requireZero(t, code, stdout)
}

func TestBinary_Investments_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "investments", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "holdings") {
		t.Errorf("investments help missing 'holdings'")
	}
	if !strings.Contains(stdout, "securities") {
		t.Errorf("investments help missing 'securities'")
	}
}

func TestBinary_Items_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "items", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("items help missing 'list'")
	}
}

func TestBinary_Liabilities_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "liabilities", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("liabilities help missing 'list'")
	}
}

func TestBinary_NetWorth_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "net-worth", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("net-worth help missing Usage:")
	}
}

func TestBinary_Recurring_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "recurring", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("recurring help missing 'list'")
	}
}

func TestBinary_Rules_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "rules", "--help")
	requireZero(t, code, stdout)
}

func TestBinary_Sync_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "sync", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("sync help missing Usage:")
	}
}

func TestBinary_Tags_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "tags", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("tags help missing 'list'")
	}
}

func TestBinary_Transactions_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "transactions", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("transactions help missing 'list'")
	}
}

func TestBinary_Tx_Alias(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "tx", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("tx alias help missing 'list'")
	}
}

func TestBinary_Link_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "link", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("link help missing Usage:")
	}
}

func TestBinary_Plaid_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "plaid", "--help")
	requireZero(t, code, stdout)
}

func TestBinary_Providers_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "providers", "--help")
	requireZero(t, code, stdout)
}

func TestBinary_Feedback_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "feedback", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("feedback help missing Usage:")
	}
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

func TestBinary_TransactionsList_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "transactions", "list", "--help")
	requireZero(t, code, stdout)
	flags := []string{"--account", "--category", "--limit", "--offset", "--needs-review", "--pending"}
	for _, flag := range flags {
		if !strings.Contains(stdout, flag) {
			t.Errorf("transactions list help missing flag %q", flag)
		}
	}
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

// ─── Coverage check ───

func TestCoverageReport(t *testing.T) {
	var uncovered []string
	for _, cmd := range requiredCommands {
		mu.Lock()
		covered := executedCmds[cmd]
		for executed := range executedCmds {
			if strings.HasPrefix(executed, cmd+" ") {
				covered = true
				break
			}
		}
		mu.Unlock()
		if !covered {
			uncovered = append(uncovered, cmd)
		}
	}

	if len(uncovered) > 0 {
		t.Errorf("%d commands have NO E2E test coverage: %v. Add a TestBinary_<command>_Help test above this one.", len(uncovered), uncovered)
	}
	t.Logf("E2E command coverage: %d/%d commands tested", len(requiredCommands)-len(uncovered), len(requiredCommands))
}
