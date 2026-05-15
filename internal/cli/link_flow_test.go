package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/thedavidweng/money/internal/providers"
	"github.com/thedavidweng/money/internal/store"
	"github.com/thedavidweng/money/internal/syncer"
)

func TestRunPlaidLinkFlowNoOpenStoresLinkedItem(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open encrypted store: %v", err)
	}
	defer db.Close()

	oldStart := startPlaidLinkSessionServer
	oldOpen := openBrowser
	t.Cleanup(func() {
		startPlaidLinkSessionServer = oldStart
		openBrowser = oldOpen
	})

	var opened bool
	startPlaidLinkSessionServer = func(linkToken string, state string, timeout time.Duration) (linkSessionServer, error) {
		if linkToken != "link-token" || state == "" {
			t.Fatalf("server input linkToken=%q state=%q", linkToken, state)
		}
		return fakeLinkSessionServer{
			url: "http://127.0.0.1:4000",
			callback: providers.LinkCallback{
				PublicToken: "public-token",
				State:       state,
			},
		}, nil
	}
	openBrowser = func(url string) error {
		opened = true
		return nil
	}

	var stdout, stderr bytes.Buffer
	state := &runtimeState{store: db, stderr: &stderr}
	if err := runPlaidLinkFlow(ctx, state, fakePlaidCLIProvider{}, plaidLinkFlowOptions{NoOpen: true}, &stdout); err != nil {
		t.Fatalf("run plaid link flow: %v", err)
	}
	if opened {
		t.Fatal("browser opened despite --no-open")
	}
	if !strings.Contains(stderr.String(), "Plaid Link URL: http://127.0.0.1:4000") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	item, err := db.GetProviderItem(ctx, "pi_cli")
	if err != nil {
		t.Fatalf("provider item not stored: %v", err)
	}
	if string(item.EncryptedAccessToken) != "access-token" {
		t.Fatalf("stored token = %q", string(item.EncryptedAccessToken))
	}
}

func TestRunPlaidLinkFlowWaitsForEnterBeforeOpeningBrowser(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open encrypted store: %v", err)
	}
	defer db.Close()

	oldStart := startPlaidLinkSessionServer
	oldOpen := openBrowser
	t.Cleanup(func() {
		startPlaidLinkSessionServer = oldStart
		openBrowser = oldOpen
	})

	startPlaidLinkSessionServer = func(linkToken string, state string, timeout time.Duration) (linkSessionServer, error) {
		return fakeLinkSessionServer{
			url: "http://127.0.0.1:4000",
			callback: providers.LinkCallback{
				PublicToken: "public-token",
				State:       state,
			},
		}, nil
	}
	var openedURL string
	openBrowser = func(url string) error {
		openedURL = url
		return nil
	}

	var stdout, stderr bytes.Buffer
	state := &runtimeState{store: db, stdin: strings.NewReader("\n"), stderr: &stderr}
	if err := runPlaidLinkFlow(ctx, state, fakePlaidCLIProvider{}, plaidLinkFlowOptions{}, &stdout); err != nil {
		t.Fatalf("run plaid link flow: %v", err)
	}
	if openedURL != "http://127.0.0.1:4000" {
		t.Fatalf("opened URL = %q", openedURL)
	}
}

func TestRunPlaidLinkFlowPassesConsentProductOptions(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open encrypted store: %v", err)
	}
	defer db.Close()

	oldStart := startPlaidLinkSessionServer
	t.Cleanup(func() { startPlaidLinkSessionServer = oldStart })
	startPlaidLinkSessionServer = func(linkToken string, state string, timeout time.Duration) (linkSessionServer, error) {
		return fakeLinkSessionServer{
			url: "http://127.0.0.1:4000",
			callback: providers.LinkCallback{
				PublicToken: "public-token",
				State:       state,
			},
		}, nil
	}

	provider := &recordingPlaidCLIProvider{}
	var stdout, stderr bytes.Buffer
	state := &runtimeState{store: db, stderr: &stderr}
	err = runPlaidLinkFlow(ctx, state, provider, plaidLinkFlowOptions{
		NoOpen:                      true,
		AdditionalConsentedProducts: "investments",
		RequiredIfSupportedProducts: "liabilities",
		OptionalProducts:            "auth",
	}, &stdout)
	if err != nil {
		t.Fatalf("run plaid link flow: %v", err)
	}
	if strings.Join(provider.request.AdditionalConsentedProducts, ",") != "investments" {
		t.Fatalf("additional consented products = %#v", provider.request.AdditionalConsentedProducts)
	}
	if strings.Join(provider.request.RequiredIfSupportedProducts, ",") != "liabilities" {
		t.Fatalf("required if supported products = %#v", provider.request.RequiredIfSupportedProducts)
	}
	if strings.Join(provider.request.OptionalProducts, ",") != "auth" {
		t.Fatalf("optional products = %#v", provider.request.OptionalProducts)
	}
}

func TestRunPlaidSandboxLinkStoresLinkedItem(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open encrypted store: %v", err)
	}
	defer db.Close()

	provider := &fakePlaidSandboxProvider{publicToken: "public-sandbox"}
	var stdout bytes.Buffer
	state := &runtimeState{store: db}
	if err := runPlaidSandboxLink(ctx, state, provider, provider, plaidSandboxLinkOptions{
		InstitutionID: "ins_56",
		Products:      "transactions,liabilities",
	}, &stdout); err != nil {
		t.Fatalf("run plaid sandbox link: %v", err)
	}
	if provider.sandboxRequest.InstitutionID != "ins_56" || strings.Join(provider.sandboxRequest.Products, ",") != "transactions,liabilities" {
		t.Fatalf("request = %#v", provider.sandboxRequest)
	}
	if provider.callback.PublicToken != "public-sandbox" {
		t.Fatalf("callback = %#v", provider.callback)
	}
	item, err := db.GetProviderItem(ctx, "pi_sandbox")
	if err != nil {
		t.Fatalf("provider item not stored: %v", err)
	}
	if string(item.EncryptedAccessToken) != "sandbox-access-token" || strings.Join(item.Products, ",") != "transactions,liabilities" {
		t.Fatalf("item = %#v", item)
	}
	if !strings.Contains(stdout.String(), "Linked plaid Sandbox Provider Item pi_sandbox") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunPlaidSandboxLinkValidation(t *testing.T) {
	db, err := store.OpenDemo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	state := &runtimeState{store: db}
	for name, opts := range map[string]plaidSandboxLinkOptions{
		"production": {Environment: "production", InstitutionID: "ins_56", Products: "transactions"},
		"balance":    {Environment: "sandbox", InstitutionID: "ins_56", Products: "balance"},
	} {
		t.Run(name, func(t *testing.T) {
			provider := &fakePlaidSandboxProvider{}
			err := runPlaidSandboxLink(context.Background(), state, provider, provider, opts, io.Discard)
			var cliErr cliError
			if !errors.As(err, &cliErr) || cliErr.category != "validation" {
				t.Fatalf("err = %#v", err)
			}
		})
	}
}

func TestRunBridgeLinkFlowNoOpenStoresLinkedItem(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open encrypted store: %v", err)
	}
	defer db.Close()

	oldOpen := openBrowser
	t.Cleanup(func() { openBrowser = oldOpen })
	var opened bool
	openBrowser = func(url string) error {
		opened = true
		return nil
	}

	var stdout bytes.Buffer
	state := &runtimeState{store: db}
	if err := runBridgeLinkFlow(ctx, state, fakeBridgeCLIProvider{}, "", true, &stdout); err != nil {
		t.Fatalf("run bridge link flow: %v", err)
	}
	if opened {
		t.Fatal("browser opened despite --no-open")
	}
	if !strings.Contains(stdout.String(), "Bridge Connect URL: https://connect.bridgeapi.io/session/session-1") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	item, err := db.GetProviderItem(ctx, "bridge:item_cli")
	if err != nil {
		t.Fatalf("provider item not stored: %v", err)
	}
	if item.ExternalUserID != "bridge-user" {
		t.Fatalf("external user id = %q", item.ExternalUserID)
	}
}

func TestSyncJSONNoLinkedItemsReturnsSuccessWarning(t *testing.T) {
	configPath := writeTestConfig(t, "")

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"--config", configPath, "sync", "--json"}, nil, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exit code = %d; stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "NO_LINKED_PROVIDER_ITEMS") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestWriteSyncJSONPartialFailureReturnsExitSentinelWithItemDiagnostics(t *testing.T) {
	var stdout bytes.Buffer
	result := syncer.Result{Items: []syncer.ItemResult{
		{Provider: "plaid", ProviderItemID: "pi_ok", Status: "ok"},
		{Provider: "bridge", ProviderItemID: "pi_bad", Status: "error", ErrorCode: "NETWORK_ERROR"},
	}}

	err := writeSyncJSON(&stdout, result, syncer.PartialFailure{Result: result})
	var exitErr cliExit
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %#v, want cliExit", err)
	}
	if exitErr.exitCode != 6 {
		t.Fatalf("exit code = %d, want 6", exitErr.exitCode)
	}
	if !strings.Contains(stdout.String(), `"ok": false`) || !strings.Contains(stdout.String(), `"pi_bad"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestSelectLinkInstitutionRequiresExplicitIDForMultipleMatches(t *testing.T) {
	institutions := []providers.Institution{
		{ID: "plaid:ins_1", ProviderInstitutionID: "ins_1", Name: "Bank One"},
		{ID: "plaid:ins_2", ProviderInstitutionID: "ins_2", Name: "Bank Two"},
	}

	_, err := selectLinkInstitution(institutions, "")
	if err == nil {
		t.Fatal("expected explicit institution id error")
	}

	selected, err := selectLinkInstitution(institutions, "ins_2")
	if err != nil {
		t.Fatalf("select institution: %v", err)
	}
	if selected.ProviderInstitutionID != "ins_2" {
		t.Fatalf("selected = %#v", selected)
	}
}

func TestSupportedProviderAvailabilityMarksMissingCredentials(t *testing.T) {
	rows := supportedProviderAvailability("plaid", []providers.ConfigDiagnostic{{
		Code:     "PROVIDER_CREDENTIALS_MISSING",
		Message:  "plaid credentials are missing.",
		Severity: "warn",
	}})

	if len(rows) != 1 {
		t.Fatalf("rows = %#v", rows)
	}
	if rows[0].Provider != "plaid" || rows[0].Status != "unavailable" || rows[0].Code != "PROVIDER_CREDENTIALS_MISSING" {
		t.Fatalf("row = %#v", rows[0])
	}
	if !strings.Contains(rows[0].Guidance, "providers.plaid credentials") {
		t.Fatalf("guidance = %q", rows[0].Guidance)
	}
}

type fakeLinkSessionServer struct {
	url      string
	callback providers.LinkCallback
}

func (s fakeLinkSessionServer) LinkURL() string {
	return s.url
}

func (s fakeLinkSessionServer) Wait(ctx context.Context) (providers.LinkCallback, error) {
	return s.callback, nil
}

func (s fakeLinkSessionServer) Shutdown(ctx context.Context) error {
	return nil
}

func TestRunPlaidLinkFlowReturnsCLIErrorOnCancel(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open encrypted store: %v", err)
	}
	defer db.Close()

	oldStart := startPlaidLinkSessionServer
	t.Cleanup(func() { startPlaidLinkSessionServer = oldStart })
	startPlaidLinkSessionServer = func(linkToken string, state string, timeout time.Duration) (linkSessionServer, error) {
		return fakeLinkSessionServer{
			url: "http://127.0.0.1:4000",
			callback: providers.LinkCallback{
				State:  state,
				Status: "cancel",
				Metadata: providers.LinkMetadata{LinkSessionID: "sess-cancel"},
			},
		}, nil
	}

	var stdout bytes.Buffer
	state := &runtimeState{store: db}
	err = runPlaidLinkFlow(ctx, state, fakePlaidCLIProvider{}, plaidLinkFlowOptions{CommandName: "providers.plaid.link", NoOpen: true}, &stdout)
	var cliErr cliError
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected cliError, got %#v", err)
	}
	if cliErr.command != "providers.plaid.link" {
		t.Fatalf("command = %q, want providers.plaid.link", cliErr.command)
	}
	if cliErr.code != "LINK_CANCELED" {
		t.Fatalf("code = %q, want LINK_CANCELED", cliErr.code)
	}
	if cliErr.category != "safety" {
		t.Fatalf("category = %q, want safety", cliErr.category)
	}
	if !cliErr.retryable {
		t.Fatal("expected retryable")
	}
	if cliErr.exitCode != 10 {
		t.Fatalf("exitCode = %d, want 10", cliErr.exitCode)
	}
	if !strings.Contains(cliErr.message, "canceled") {
		t.Fatalf("message = %q, want 'canceled'", cliErr.message)
	}
}

func TestRunPlaidLinkFlowReturnsCLIErrorOnLinkError(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open encrypted store: %v", err)
	}
	defer db.Close()

	oldStart := startPlaidLinkSessionServer
	t.Cleanup(func() { startPlaidLinkSessionServer = oldStart })
	startPlaidLinkSessionServer = func(linkToken string, state string, timeout time.Duration) (linkSessionServer, error) {
		return fakeLinkSessionServer{
			url: "http://127.0.0.1:4000",
			callback: providers.LinkCallback{
				State:  state,
				Status: "error",
				Error:  providers.LinkError{Type: "INSTITUTION_ERROR", Code: "INSUFFICIENT_CREDENTIALS", Message: "user entered invalid credentials"},
				Metadata: providers.LinkMetadata{RequestID: "req-123", LinkSessionID: "sess-456"},
			},
		}, nil
	}

	var stdout bytes.Buffer
	state := &runtimeState{store: db}
	err = runPlaidLinkFlow(ctx, state, fakePlaidCLIProvider{}, plaidLinkFlowOptions{CommandName: "link", NoOpen: true}, &stdout)
	var cliErr cliError
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected cliError, got %#v", err)
	}
	if cliErr.command != "link" {
		t.Fatalf("command = %q, want link", cliErr.command)
	}
	if cliErr.code != "LINK_ERROR" {
		t.Fatalf("code = %q, want LINK_ERROR", cliErr.code)
	}
	if cliErr.category != "api" {
		t.Fatalf("category = %q, want api", cliErr.category)
	}
	if cliErr.retryable {
		t.Fatal("expected not retryable")
	}
	if cliErr.exitCode != 6 {
		t.Fatalf("exitCode = %d, want 6", cliErr.exitCode)
	}
	if !strings.Contains(cliErr.message, "INSUFFICIENT_CREDENTIALS") {
		t.Fatalf("message = %q", cliErr.message)
	}
}

func TestRunPlaidLinkFlowWritesJSONWhenStateJSON(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open encrypted store: %v", err)
	}
	defer db.Close()

	oldStart := startPlaidLinkSessionServer
	t.Cleanup(func() { startPlaidLinkSessionServer = oldStart })
	startPlaidLinkSessionServer = func(linkToken string, state string, timeout time.Duration) (linkSessionServer, error) {
		return fakeLinkSessionServer{
			url: "http://127.0.0.1:4000",
			callback: providers.LinkCallback{
				PublicToken: "public-token",
				State:       state,
			},
		}, nil
	}

	var stdout bytes.Buffer
	state := &runtimeState{store: db, json: true}
	if err := runPlaidLinkFlow(ctx, state, fakePlaidCLIProvider{}, plaidLinkFlowOptions{CommandName: "link", NoOpen: true}, &stdout); err != nil {
		t.Fatalf("run plaid link flow: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, `"ok": true`) {
		t.Fatalf("expected success envelope, stdout = %q", out)
	}
	if !strings.Contains(out, `"provider_item_id": "pi_cli"`) {
		t.Fatalf("expected provider_item_id in JSON, stdout = %q", out)
	}
	if !strings.Contains(out, `"institution_id": "inst_cli"`) {
		t.Fatalf("expected institution_id in JSON, stdout = %q", out)
	}
	if strings.Contains(out, "Plaid Link URL") {
		t.Fatal("unexpected progress text in JSON mode")
	}
	if strings.Contains(out, "Linked plaid") {
		t.Fatal("unexpected plain text output in JSON mode")
	}
}

func TestRunPlaidSandboxLinkWritesJSONWhenStateJSON(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open encrypted store: %v", err)
	}
	defer db.Close()

	provider := &fakePlaidSandboxProvider{publicToken: "public-sandbox"}
	var stdout bytes.Buffer
	state := &runtimeState{store: db, json: true}
	if err := runPlaidSandboxLink(ctx, state, provider, provider, plaidSandboxLinkOptions{
		InstitutionID: "ins_56",
		Products:      "transactions",
	}, &stdout); err != nil {
		t.Fatalf("run plaid sandbox link: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, `"ok": true`) {
		t.Fatalf("expected success envelope, stdout = %q", out)
	}
	if !strings.Contains(out, `"provider_item_id": "pi_sandbox"`) {
		t.Fatalf("expected provider_item_id in JSON, stdout = %q", out)
	}
	if !strings.Contains(out, `"institution_id": "plaid:ins_56"`) {
		t.Fatalf("expected institution_id in JSON, stdout = %q", out)
	}
	if strings.Contains(out, "Linked plaid Sandbox") {
		t.Fatal("unexpected plain text output in JSON mode")
	}
}

type fakePlaidCLIProvider struct{}

func (fakePlaidCLIProvider) Name() string { return "plaid" }
func (fakePlaidCLIProvider) ValidateConfig(ctx context.Context) []providers.ConfigDiagnostic {
	return nil
}
func (fakePlaidCLIProvider) SearchInstitutions(ctx context.Context, query string) ([]providers.Institution, error) {
	return nil, nil
}
func (fakePlaidCLIProvider) CreateLinkSession(ctx context.Context, request providers.LinkRequest) (providers.LinkSession, error) {
	return providers.LinkSession{Provider: "plaid", LinkToken: "link-token", State: request.State}, nil
}
func (fakePlaidCLIProvider) ExchangeLinkToken(ctx context.Context, session providers.LinkSession, callback providers.LinkCallback) (providers.LinkedItem, error) {
	return providers.LinkedItem{
		Institution: providers.Institution{
			ID:                    "inst_cli",
			Name:                  "CLI Bank",
			Provider:              "plaid",
			ProviderInstitutionID: "ins_cli",
		},
		ProviderItem: providers.ProviderItem{
			ID:                     "pi_cli",
			Provider:               "plaid",
			InstitutionID:          "inst_cli",
			ProviderExternalItemID: "item_cli",
			EncryptedAccessToken:   []byte("access-token"),
			Status:                 "active",
		},
	}, nil
}
func (fakePlaidCLIProvider) Sync(ctx context.Context, item providers.ProviderItem, sink providers.SyncSink) (providers.SyncResult, error) {
	return providers.SyncResult{}, nil
}

type recordingPlaidCLIProvider struct {
	fakePlaidCLIProvider
	request providers.LinkRequest
}

func (p *recordingPlaidCLIProvider) CreateLinkSession(ctx context.Context, request providers.LinkRequest) (providers.LinkSession, error) {
	p.request = request
	return providers.LinkSession{Provider: "plaid", LinkToken: "link-token", State: request.State}, nil
}

type fakePlaidSandboxProvider struct {
	fakePlaidCLIProvider
	publicToken    string
	sandboxRequest providers.SandboxPublicTokenRequest
	callback       providers.LinkCallback
}

func (p *fakePlaidSandboxProvider) CreateSandboxPublicToken(ctx context.Context, request providers.SandboxPublicTokenRequest) (string, error) {
	p.sandboxRequest = request
	return p.publicToken, nil
}

func (p *fakePlaidSandboxProvider) ExchangeLinkToken(ctx context.Context, session providers.LinkSession, callback providers.LinkCallback) (providers.LinkedItem, error) {
	p.callback = callback
	return providers.LinkedItem{
		Institution: providers.Institution{
			ID:                    "plaid:ins_56",
			Name:                  "Plaid Sandbox",
			Provider:              "plaid",
			ProviderInstitutionID: "ins_56",
		},
		ProviderItem: providers.ProviderItem{
			ID:                     "pi_sandbox",
			Provider:               "plaid",
			InstitutionID:          "plaid:ins_56",
			ProviderExternalItemID: "item_sandbox",
			EncryptedAccessToken:   []byte("sandbox-access-token"),
			Status:                 "active",
			Products:               session.Products,
		},
	}, nil
}

type fakeBridgeCLIProvider struct{}

func (fakeBridgeCLIProvider) Name() string { return "bridge" }
func (fakeBridgeCLIProvider) ValidateConfig(ctx context.Context) []providers.ConfigDiagnostic {
	return nil
}
func (fakeBridgeCLIProvider) SearchInstitutions(ctx context.Context, query string) ([]providers.Institution, error) {
	return nil, nil
}
func (fakeBridgeCLIProvider) CreateLinkSession(ctx context.Context, request providers.LinkRequest) (providers.LinkSession, error) {
	return providers.LinkSession{
		Provider:            "bridge",
		URL:                 "https://connect.bridgeapi.io/session/session-1",
		State:               "bridge-user",
		ProviderAccessToken: "bridge-user-token",
	}, nil
}
func (fakeBridgeCLIProvider) ExchangeLinkToken(ctx context.Context, session providers.LinkSession, callback providers.LinkCallback) (providers.LinkedItem, error) {
	return providers.LinkedItem{
		Institution: providers.Institution{
			ID:                    "bridge:bank_cli",
			Name:                  "Bridge Bank",
			Provider:              "bridge",
			ProviderInstitutionID: "bank_cli",
		},
		ProviderItem: providers.ProviderItem{
			ID:                     "bridge:item_cli",
			Provider:               "bridge",
			InstitutionID:          "bridge:bank_cli",
			ProviderExternalItemID: "item_cli",
			EncryptedAccessToken:   []byte("bridge-user"),
			ExternalUserID:         "bridge-user",
			Status:                 "active",
		},
	}, nil
}
func (fakeBridgeCLIProvider) Sync(ctx context.Context, item providers.ProviderItem, sink providers.SyncSink) (providers.SyncResult, error) {
	return providers.SyncResult{}, nil
}
