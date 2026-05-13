package cli

import (
	"bufio"
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/money/internal/config"
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

	stdin := strings.NewReader("3\n")
	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      stdin,
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
	if !strings.Contains(stdout.String(), "Skip for now") {
		t.Fatal("should show skip option")
	}
	if !strings.Contains(stdout.String(), "You can always run") {
		t.Fatal("should show skip confirmation")
	}
}

func TestRunSetupWizardInvalidChoice(t *testing.T) {
	dir := t.TempDir()
	result, err := config.Setup(filepath.Join(dir, "config.yaml"), "default", false)
	if err != nil {
		t.Fatal(err)
	}

	// With only 1 unconfigured provider, skip is option 2.
	stdin := strings.NewReader("abc\n2\n")
	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      stdin,
	}
	diags := []Diagnostic{
		{Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn", Message: "plaid is not configured"},
	}
	var stdout bytes.Buffer
	err = runSetupWizard(context.Background(), state, &stdout, diags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Invalid choice") {
		t.Fatal("should show invalid choice message")
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

	// Select 1 (plaid), press Enter at browser prompt, enter client-id, enter secret, enter n (don't continue)
	stdin := strings.NewReader("1\n\nmy-client-id\nmy-secret\nn\n")
	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      stdin,
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

func TestRunSetupWizardConfigureAllProviders(t *testing.T) {
	origOpenBrowser := openBrowser
	openBrowser = func(url string) error { return nil }
	defer func() { openBrowser = origOpenBrowser }()

	dir := t.TempDir()
	result, err := config.Setup(filepath.Join(dir, "config.yaml"), "default", false)
	if err != nil {
		t.Fatal(err)
	}

	// Select 1 (plaid), enter credentials, y (continue), 1 (bridge), enter credentials, n
	stdin := strings.NewReader("1\n\nplaid-id\nplaid-secret\ny\n1\n\nbridge-id\nbridge-secret\nn\n")
	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      stdin,
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
