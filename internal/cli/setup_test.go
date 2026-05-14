package cli

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/plaidlogin"
	"github.com/thedavidweng/money/internal/prompt"
)

// promptForProviderCredentials tests

func TestPromptForProviderCredentialsAllPrefilled(t *testing.T) {
	origOpenBrowser := openBrowser
	openBrowser = func(url string) error { return nil }
	defer func() { openBrowser = origOpenBrowser }()

	var stdout bytes.Buffer
	stdin := strings.NewReader("")
	secrets := map[string]string{
		"client_id": "already-set",
		"secret":    "already-set",
	}
	spec := config.PlaidSpec

	err := promptForProviderCredentials(&stdout, bufio.NewReader(stdin), spec, secrets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secrets["client_id"] != "already-set" || secrets["secret"] != "already-set" {
		t.Fatal("secrets were unexpectedly modified")
	}
	if strings.Contains(stdout.String(), "client-id:") {
		t.Fatal("should not prompt when all fields are pre-filled")
	}
	if strings.Contains(stdout.String(), "Press Enter") {
		t.Fatal("should not show Press Enter prompt when all fields are pre-filled")
	}
}

func TestPromptForProviderCredentialsMissingFields(t *testing.T) {
	origOpenBrowser := openBrowser
	openBrowser = func(url string) error { return nil }
	defer func() { openBrowser = origOpenBrowser }()

	var stdout bytes.Buffer
	stdin := strings.NewReader("\nmy-client-id\nmy-secret\n")
	secrets := map[string]string{}
	spec := config.PlaidSpec

	err := promptForProviderCredentials(&stdout, bufio.NewReader(stdin), spec, secrets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secrets["client_id"] != "my-client-id" {
		t.Fatalf("client_id = %q, want my-client-id", secrets["client_id"])
	}
	if secrets["secret"] != "my-secret" {
		t.Fatalf("secret = %q, want my-secret", secrets["secret"])
	}
}

func TestPromptForProviderCredentialsEmptyInputRetry(t *testing.T) {
	origOpenBrowser := openBrowser
	openBrowser = func(url string) error { return nil }
	defer func() { openBrowser = origOpenBrowser }()

	var stdout bytes.Buffer
	stdin := strings.NewReader("\n\nvalid-id\nvalid-secret\n")
	secrets := map[string]string{}
	spec := config.PlaidSpec

	err := promptForProviderCredentials(&stdout, bufio.NewReader(stdin), spec, secrets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secrets["client_id"] != "valid-id" {
		t.Fatalf("client_id = %q, want valid-id", secrets["client_id"])
	}
	if secrets["secret"] != "valid-secret" {
		t.Fatalf("secret = %q, want valid-secret", secrets["secret"])
	}
	if !strings.Contains(stdout.String(), "is required") {
		t.Fatal("should show retry message for empty input")
	}
}

func TestPromptForProviderCredentialsNoHelpURL(t *testing.T) {
	origOpenBrowser := openBrowser
	openBrowser = func(url string) error { return nil }
	defer func() { openBrowser = origOpenBrowser }()

	var stdout bytes.Buffer
	stdin := strings.NewReader("some-id\nsome-secret\n")
	secrets := map[string]string{}
	spec := config.ProviderSpec{
		Name:         "test-provider",
		SecretFields: []string{"client_id", "secret"},
		HelpURL:      "",
	}

	err := promptForProviderCredentials(&stdout, bufio.NewReader(stdin), spec, secrets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(stdout.String(), "Open") {
		t.Fatal("should not show URL when HelpURL is empty")
	}
	if secrets["client_id"] != "some-id" {
		t.Fatalf("client_id = %q, want some-id", secrets["client_id"])
	}
}

// runSetupWizard tests

func TestRunSetupWizardNoUnconfigured(t *testing.T) {
	state := &runtimeState{
		configPath: "",
		profile:    "default",
		stdin:      strings.NewReader(""),
	}
	diags := []Diagnostic{
		{Section: "Config", Code: "CONFIG_OK", Status: "ok"},
		{Section: "Providers", Code: "PROVIDER_CONFIGURED", Status: "ok", Message: "plaid configured"},
	}
	var stdout bytes.Buffer
	err := runSetupWizard(context.Background(), state, &stdout, diags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.String() != "" {
		t.Fatalf("expected no output, got: %s", stdout.String())
	}
}

func TestRunSetupWizardSkip(t *testing.T) {
	dir := t.TempDir()
	result, err := config.Setup(filepath.Join(dir, "config.yaml"), "default", false)
	if err != nil {
		t.Fatal(err)
	}

	stdin := strings.NewReader("")
	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      stdin,
		prompter:   prompt.NewFake("skip"),
	}
	diags := []Diagnostic{
		{Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn", Message: "plaid is not configured"},
		{Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn", Message: "bridge is not configured"},
	}
	var stdout bytes.Buffer
	err = runSetupWizard(context.Background(), state, &stdout, diags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "You can always run") {
		t.Fatal("should show skip confirmation")
	}
}

func TestRunSetupWizardPromptFailure(t *testing.T) {
	dir := t.TempDir()
	result, err := config.Setup(filepath.Join(dir, "config.yaml"), "default", false)
	if err != nil {
		t.Fatal(err)
	}

	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      strings.NewReader(""),
		prompter:   prompt.NewFake("missing"),
	}
	diags := []Diagnostic{
		{Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn", Message: "plaid is not configured"},
	}
	var stdout bytes.Buffer
	err = runSetupWizard(context.Background(), state, &stdout, diags)
	if err == nil {
		t.Fatal("expected prompt failure")
	}
}

func TestRunSetupWizardSelectPlaidAndConfigure(t *testing.T) {
	origOpenBrowser := openBrowser
	openBrowser = func(url string) error { return nil }
	defer func() { openBrowser = origOpenBrowser }()

	dir := t.TempDir()
	result, err := config.Setup(filepath.Join(dir, "config.yaml"), "default", false)
	if err != nil {
		t.Fatal(err)
	}

	// Select plaid, press Enter at browser prompt, enter client-id, enter secret, enter n (don't continue)
	stdin := strings.NewReader("\nmy-client-id\nmy-secret\nn\n")
	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      stdin,
		prompter:   prompt.NewFake("plaid", "manual"),
	}
	diags := []Diagnostic{
		{Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn", Message: "plaid is not configured"},
		{Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn", Message: "bridge is not configured"},
	}
	var stdout bytes.Buffer
	err = runSetupWizard(context.Background(), state, &stdout, diags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := config.Load(config.Options{ConfigPath: result.ConfigPath, Profile: "default"})
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if len(cfg.Providers["plaid"].Fields) == 0 {
		t.Fatal("plaid provider should be configured")
	}

	if !strings.Contains(stdout.String(), "plaid configured") {
		t.Fatalf("should show plaid configured message, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Would you like to configure another provider?") {
		t.Fatalf("should ask to continue, got: %s", stdout.String())
	}
}

func TestRunSetupWizardSelectPlaidAndSkipMethodWritesNothing(t *testing.T) {
	dir := t.TempDir()
	result, err := config.Setup(filepath.Join(dir, "config.yaml"), "default", false)
	if err != nil {
		t.Fatal(err)
	}
	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      strings.NewReader(""),
		prompter:   prompt.NewFake("plaid", "skip"),
	}
	diags := []Diagnostic{
		{Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn", Message: "plaid is not configured"},
	}
	var stdout bytes.Buffer
	if err := runSetupWizard(context.Background(), state, &stdout, diags); err != nil {
		t.Fatalf("runSetupWizard: %v", err)
	}
	cfg, err := config.Load(config.Options{ConfigPath: result.ConfigPath, Profile: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Providers["plaid"]; ok {
		t.Fatal("plaid provider was written after skip")
	}
}

func TestRunSetupWizardSelectPlaidDashboardLogin(t *testing.T) {
	oldRunPlaidLogin := runPlaidLoginCLI
	t.Cleanup(func() { runPlaidLoginCLI = oldRunPlaidLogin })
	var called bool
	runPlaidLoginCLI = func(ctx context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer, opts plaidLoginCLIOptions) error {
		called = true
		if opts.CommandName != "plaid.login" || opts.Environment != "sandbox" {
			t.Fatalf("opts = %#v", opts)
		}
		return writePlaidLoginResult(state, stdout, plaidlogin.LoginResult{
			Provider:          "plaid",
			TeamID:            "team_1",
			Environment:       "sandbox",
			KeysWritten:       2,
			CredentialAction:  "written",
			DashboardAuthPath: filepath.Join(filepath.Dir(state.configPath), "plaid-dashboard-auth.json"),
			NextCommand:       "money link <institution-query>",
			ConfigPath:        state.configPath,
			EnvPath:           filepath.Join(filepath.Dir(state.configPath), ".env"),
		}, opts.CommandName)
	}
	dir := t.TempDir()
	result, err := config.Setup(filepath.Join(dir, "config.yaml"), "default", false)
	if err != nil {
		t.Fatal(err)
	}
	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      strings.NewReader(""),
		prompter:   prompt.NewFake("plaid", "dashboard"),
	}
	diags := []Diagnostic{
		{Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn", Message: "plaid is not configured"},
	}
	var stdout bytes.Buffer
	if err := runSetupWizard(context.Background(), state, &stdout, diags); err != nil {
		t.Fatalf("runSetupWizard: %v", err)
	}
	if !called {
		t.Fatal("dashboard login was not called")
	}
	if !strings.Contains(stdout.String(), "Plaid Dashboard login complete") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestRunInteractiveProviderConfigureRequiresForceOrHumanConfirmationForExistingCredentials(t *testing.T) {
	dir := t.TempDir()
	result, err := config.Setup(filepath.Join(dir, "config.yaml"), "default", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := config.ConfigureProvider(result.ConfigPath, "default", config.PlaidSpec, map[string]string{
		"client_id": "existing-client",
		"secret":    "existing-secret",
	}, map[string]string{"environment": "sandbox"}, false); err != nil {
		t.Fatal(err)
	}

	cmd := newConfigureCommand(&runtimeState{}, io.Discard)
	cmd.SetArgs([]string{"plaid", "--client-id", "new-client", "--secret", "new-secret"})
	if err := cmd.ParseFlags([]string{"plaid", "--client-id", "new-client", "--secret", "new-secret"}); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	err = runInteractiveProviderConfigure(&runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		json:       true,
	}, &stdout, "plaid", cmd, nil)
	cliErr, ok := err.(cliError)
	if !ok || cliErr.code != "CONFIRMATION_REQUIRED" || cliErr.exitCode != 10 {
		t.Fatalf("err = %#v", err)
	}

	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      strings.NewReader(""),
		prompter:   prompt.NewFake("yes"),
	}
	if err := runInteractiveProviderConfigure(state, &stdout, "plaid", cmd, nil); err != nil {
		t.Fatalf("human confirmed configure: %v", err)
	}
	cfg, err := config.Load(config.Options{ConfigPath: result.ConfigPath, Profile: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Providers["plaid"].Fields["client_id"] != "new-client" || cfg.Providers["plaid"].Fields["secret"] != "new-secret" {
		t.Fatalf("fields = %#v", cfg.Providers["plaid"].Fields)
	}
}

func TestRunSetupWizardConfigureAllProviders(t *testing.T) {
	origOpenBrowser := openBrowser
	openBrowser = func(url string) error { return nil }
	defer func() { openBrowser = origOpenBrowser }()

	dir := t.TempDir()
	result, err := config.Setup(filepath.Join(dir, "config.yaml"), "default", false)
	if err != nil {
		t.Fatal(err)
	}

	// Select plaid, enter credentials, continue, select bridge, enter credentials.
	stdin := strings.NewReader("\nplaid-id\nplaid-secret\ny\n\nbridge-id\nbridge-secret\nn\n")
	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      stdin,
		prompter:   prompt.NewFake("plaid", "manual", "bridge"),
	}
	diags := []Diagnostic{
		{Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn", Message: "plaid is not configured"},
		{Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn", Message: "bridge is not configured"},
	}
	var stdout bytes.Buffer
	err = runSetupWizard(context.Background(), state, &stdout, diags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := config.Load(config.Options{ConfigPath: result.ConfigPath, Profile: "default"})
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if len(cfg.Providers["plaid"].Fields) == 0 {
		t.Fatal("plaid should be configured")
	}
	if len(cfg.Providers["bridge"].Fields) == 0 {
		t.Fatal("bridge should be configured")
	}

	if !strings.Contains(stdout.String(), "All providers configured") {
		t.Fatalf("should show all done message, got: %s", stdout.String())
	}
}
