package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/core"
	"github.com/thedavidweng/money/internal/plaidlogin"
	"github.com/thedavidweng/money/internal/prompt"
	"github.com/thedavidweng/money/internal/store"
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
	runPlaidLoginCLI = func(ctx context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer, opts *plaidLoginCLIOptions) error {
		called = true
		if opts.CommandName != "plaid.login" || opts.Environment != "sandbox" {
			t.Fatalf("opts = %#v", opts)
		}
		if _, err := fmt.Fprintln(stderr, "oauth progress"); err != nil {
			return err
		}
		return writePlaidLoginResult(state, stdout, &plaidlogin.LoginResult{
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
	var stderr bytes.Buffer
	state := &runtimeState{
		configPath: result.ConfigPath,
		profile:    "default",
		stdin:      strings.NewReader(""),
		stderr:     &stderr,
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
	if strings.Contains(stdout.String(), "oauth progress") {
		t.Fatalf("OAuth progress should not be written to setup stdout: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "oauth progress") {
		t.Fatalf("OAuth progress missing from setup stderr: %s", stderr.String())
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
	cliErr, ok := err.(*cliError)
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

// Links diagnostics tests

func TestAppendLinksDiagnosticsNoItems(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	diags := appendLinksDiagnostics(ctx, db)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Section != "Links" {
		t.Fatalf("section = %q", diags[0].Section)
	}
	if diags[0].Status != "ok" {
		t.Fatalf("status = %q, want ok", diags[0].Status)
	}
}

func TestAppendLinksDiagnosticsWithItems(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Demo already has plaid items. Add a bridge item.
	if err := db.StoreLinkedProviderItem(ctx, &store.LinkedProviderItem{
		Institution: store.LinkedInstitution{ID: "inst_bridge", Name: "Bridge Bank", Provider: "bridge", ProviderInstitutionID: "ins_bridge"},
		Item: store.LinkedItem{
			ID: "pi_bridge", Provider: "bridge", InstitutionID: "inst_bridge",
			ProviderExternalItemID: "item_bridge", EncryptedAccessToken: []byte("tok"),
			Status: "active", Products: []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store bridge item: %v", err)
	}

	diags := appendLinksDiagnostics(ctx, db)
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics (bridge + plaid), got %d", len(diags))
	}
	for _, d := range diags {
		if d.Section != "Links" {
			t.Fatalf("section = %q", d.Section)
		}
		if d.Status != "ok" {
			t.Fatalf("status = %q, want ok", d.Status)
		}
	}
}

// Sync diagnostics tests

func TestAppendSyncDiagnosticsNoItemsNoRuns(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// No provider items in clean demo → no sync diagnostics expected.
	diags := appendSyncDiagnostics(ctx, db)
	// Demo has provider items, so we expect a warning about no sync runs.
	hasWarn := false
	for _, d := range diags {
		if d.Section == "Sync" && d.Status == "warn" {
			hasWarn = true
		}
	}
	if !hasWarn {
		t.Fatalf("expected sync warn diagnostic, got: %+v", diags)
	}
}

func TestAppendSyncDiagnosticsWithRecentRun(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Record a successful sync run for the demo plaid item.
	if err := db.RecordSyncRun(ctx, &core.SyncRun{
		Provider:       "plaid",
		ProviderItemID: "pi_demo_plaid",
		StartedAt:      "2026-05-10T10:00:00Z",
		FinishedAt:     "2026-05-10T10:00:02Z",
		Status:         "ok",
	}); err != nil {
		t.Fatalf("record sync run: %v", err)
	}

	diags := appendSyncDiagnostics(ctx, db)
	hasOK := false
	for _, d := range diags {
		if d.Section == "Sync" && d.Status == "ok" {
			hasOK = true
			if !strings.Contains(d.Message, "ok") {
				t.Fatalf("expected ok in message, got %q", d.Message)
			}
		}
	}
	if !hasOK {
		t.Fatalf("expected sync ok diagnostic, got: %+v", diags)
	}
}

// Doctor fix tests

func TestRunDoctorFixLinksRemovesErroredItems(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add an errored provider item.
	if err := db.StoreLinkedProviderItem(ctx, &store.LinkedProviderItem{
		Institution: store.LinkedInstitution{ID: "inst_err", Name: "Err Bank", Provider: "plaid", ProviderInstitutionID: "ins_err"},
		Item: store.LinkedItem{
			ID: "pi_err", Provider: "plaid", InstitutionID: "inst_err",
			ProviderExternalItemID: "item_err", EncryptedAccessToken: []byte("tok"),
			Status: "error", Products: []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store errored item: %v", err)
	}

	fixed, err := runDoctorFixLinks(ctx, db)
	if err != nil {
		t.Fatalf("fix links: %v", err)
	}
	if fixed != 1 {
		t.Fatalf("expected 1 fixed, got %d", fixed)
	}

	// Verify the errored item is gone.
	items, err := db.ListProviderItems(ctx, store.ProviderItemQuery{})
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	for _, item := range items {
		if item.ID == "pi_err" {
			t.Fatal("errored item should have been removed")
		}
	}
}

func TestRunDoctorFixLinksSkipsActiveItems(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Demo has active plaid items — fix should not remove them.
	fixed, err := runDoctorFixLinks(ctx, db)
	if err != nil {
		t.Fatalf("fix links: %v", err)
	}
	if fixed != 0 {
		t.Fatalf("expected 0 fixed, got %d", fixed)
	}
}

func TestRunDoctorFixSyncMarksStuckRuns(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// RecordSyncRun with empty finished_at produces a stuck run (NULL finished_at).
	if err := db.RecordSyncRun(ctx, &core.SyncRun{
		Provider:       "plaid",
		ProviderItemID: "pi_demo_plaid",
		StartedAt:      "2026-05-10T10:00:00Z",
		FinishedAt:     "",
		Status:         "ok",
	}); err != nil {
		t.Fatalf("record stuck run: %v", err)
	}

	fixed, err := runDoctorFixSync(ctx, db)
	if err != nil {
		t.Fatalf("fix sync: %v", err)
	}
	if fixed != 1 {
		t.Fatalf("expected 1 fixed, got %d", fixed)
	}
}

// Structured logging tests

func TestDoctorDiagnosticsLogToSlog(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	diags := appendLinksDiagnostics(ctx, db)
	logDiagnostics(logger, diags)

	output := buf.String()
	if !strings.Contains(output, "Links") {
		t.Fatalf("expected Links in slog output, got: %s", output)
	}
	if !strings.Contains(output, "LINKS_OK") {
		t.Fatalf("expected LINKS_OK in slog output, got: %s", output)
	}
}

func TestDoctorSyncDiagnosticsLogToSlog(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	diags := appendSyncDiagnostics(ctx, db)
	logDiagnostics(logger, diags)

	output := buf.String()
	if !strings.Contains(output, "Sync") {
		t.Fatalf("expected Sync in slog output, got: %s", output)
	}
}

func TestDoctorFixLogsToSlog(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Add an errored item to trigger a fix.
	if err := db.StoreLinkedProviderItem(ctx, &store.LinkedProviderItem{
		Institution: store.LinkedInstitution{ID: "inst_err2", Name: "Err Bank", Provider: "plaid", ProviderInstitutionID: "ins_err2"},
		Item: store.LinkedItem{
			ID: "pi_err2", Provider: "plaid", InstitutionID: "inst_err2",
			ProviderExternalItemID: "item_err2", EncryptedAccessToken: []byte("tok"),
			Status: "error", Products: []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store errored item: %v", err)
	}

	logFixResult(logger, "links", 1)

	output := buf.String()
	if !strings.Contains(output, "doctor fix") {
		t.Fatalf("expected 'doctor fix' in slog output, got: %s", output)
	}
	if !strings.Contains(output, "links") {
		t.Fatalf("expected 'links' in slog output, got: %s", output)
	}
}
