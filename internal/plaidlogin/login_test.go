package plaidlogin

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/prompt"
)

func TestRunLoginWritesFetchedKeysAndDashboardAuth(t *testing.T) {
	configPath, envPath := writeLoginConfig(t, "")
	var callbackURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			callbackURL = r.Form.Get("redirect_uri")
			w.Write([]byte(`{"access_token":"dashboard-access","refresh_token":"dashboard-refresh","expires_in":3600,"token_type":"Bearer"}`))
		case "/cli/teams/list":
			w.Write([]byte(`{"teams":[{"team_id":"team_1","client_id":"client_1","company":"Acme"}]}`))
		case "/cli/keys/fetch":
			w.Write([]byte(`{"client_id":"client_1","secrets":{"sandbox":["sandbox-secret"]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	result, err := RunLogin(context.Background(), LoginOptions{
		ConfigPath:   configPath,
		Profile:      "default",
		Environment:  "sandbox",
		CallbackCode: "auth-code",
		RedirectPort: 49152,
		CodeVerifier: "verifier",
		State:        "state",
		HTTPClient:   server.Client(),
		TokenURL:     server.URL + "/oauth/token",
		DashboardURL: server.URL,
		Now:          func() time.Time { return time.Unix(100, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("RunLogin: %v", err)
	}
	if result.KeysWritten != 2 || result.CredentialAction != "written" || result.TeamID != "team_1" {
		t.Fatalf("result = %#v", result)
	}
	if callbackURL != "http://localhost:49152/oauth/callback" {
		t.Fatalf("redirect_uri = %q", callbackURL)
	}
	envContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(envContent), "PLAID_CLIENT_ID=client_1") || !strings.Contains(string(envContent), "PLAID_SECRET=sandbox-secret") {
		t.Fatalf("env = %s", string(envContent))
	}
	auth, err := ReadAuthFile(filepath.Join(filepath.Dir(configPath), "plaid-dashboard-auth.json"))
	if err != nil {
		t.Fatalf("auth file: %v", err)
	}
	if auth.TeamID != "team_1" || auth.ClientID != "client_1" || auth.AccessToken != "dashboard-access" {
		t.Fatalf("auth = %#v", auth)
	}
}

func TestRunLoginCanRepairMissingPlaidEnvRefs(t *testing.T) {
	configPath, _ := writeLoginConfig(t, `
providers:
  plaid:
    client_id:
      env: PLAID_CLIENT_ID
    secret:
      env: PLAID_SECRET
    environment: sandbox
`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			w.Write([]byte(`{"access_token":"a","refresh_token":"r","expires_in":3600,"token_type":"Bearer"}`))
		case "/cli/teams/list":
			w.Write([]byte(`{"teams":[{"team_id":"team_1","client_id":"client_1","company":"Acme"}]}`))
		case "/cli/keys/fetch":
			w.Write([]byte(`{"client_id":"client_1","secrets":{"sandbox":["sandbox-secret"]}}`))
		}
	}))
	defer server.Close()
	_, err := RunLogin(context.Background(), LoginOptions{
		ConfigPath:   configPath,
		Environment:  "sandbox",
		CallbackCode: "auth-code",
		RedirectPort: 49152,
		CodeVerifier: "verifier",
		State:        "state",
		HTTPClient:   server.Client(),
		TokenURL:     server.URL + "/oauth/token",
		DashboardURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunLogin should repair missing Plaid env refs: %v", err)
	}
}

func TestRunLoginPreservesExistingSameEnvironmentKeys(t *testing.T) {
	configPath, envPath := writeLoginConfig(t, `
providers:
  plaid:
    client_id:
      env: PLAID_CLIENT_ID
    secret:
      env: PLAID_SECRET
    environment: sandbox
`)
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\nPLAID_CLIENT_ID=existing-client\nPLAID_SECRET=existing-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := loginFakeDashboard(t, "existing-client", "existing-secret")
	defer server.Close()
	result, err := RunLogin(context.Background(), LoginOptions{
		ConfigPath:   configPath,
		Environment:  "sandbox",
		CallbackCode: "auth-code",
		RedirectPort: 49152,
		CodeVerifier: "verifier",
		State:        "state",
		HTTPClient:   server.Client(),
		TokenURL:     server.URL + "/oauth/token",
		DashboardURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunLogin should preserve same-environment keys: %v", err)
	}
	if result.CredentialAction != "preserved_existing" || result.KeysWritten != 0 {
		t.Fatalf("result = %#v", result)
	}
	envContent, _ := os.ReadFile(envPath)
	if !strings.Contains(string(envContent), "PLAID_CLIENT_ID=existing-client") || !strings.Contains(string(envContent), "PLAID_SECRET=existing-secret") {
		t.Fatalf("env changed:\n%s", string(envContent))
	}
}

func TestRunLoginOverwritesRotatedSecretForSameTeamAndEnvironment(t *testing.T) {
	configPath, envPath := writeLoginConfig(t, `
providers:
  plaid:
    client_id:
      env: PLAID_CLIENT_ID
    secret:
      env: PLAID_SECRET
    environment: sandbox
`)
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\nPLAID_CLIENT_ID=existing-client\nPLAID_SECRET=existing-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := loginFakeDashboard(t, "existing-client", "rotated-secret")
	defer server.Close()
	result, err := RunLogin(context.Background(), LoginOptions{
		ConfigPath:   configPath,
		Environment:  "sandbox",
		CallbackCode: "auth-code",
		RedirectPort: 49152,
		CodeVerifier: "verifier",
		State:        "state",
		HTTPClient:   server.Client(),
		TokenURL:     server.URL + "/oauth/token",
		DashboardURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunLogin: %v", err)
	}
	if result.CredentialAction != "written" || result.KeysWritten != 2 {
		t.Fatalf("result = %#v", result)
	}
	envContent, _ := os.ReadFile(envPath)
	if !strings.Contains(string(envContent), "PLAID_SECRET=rotated-secret") {
		t.Fatalf("env was not updated:\n%s", string(envContent))
	}
}

func TestRunLoginRejectsMismatchedTeamCredentialsWithoutForce(t *testing.T) {
	configPath, envPath := writeLoginConfig(t, `
providers:
  plaid:
    client_id:
      env: PLAID_CLIENT_ID
    secret:
      env: PLAID_SECRET
    environment: sandbox
`)
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\nPLAID_CLIENT_ID=team-a-client\nPLAID_SECRET=team-a-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := loginFakeDashboard(t, "team-b-client", "team-b-secret")
	defer server.Close()
	_, err := RunLogin(context.Background(), LoginOptions{
		ConfigPath:   configPath,
		Environment:  "sandbox",
		CallbackCode: "auth-code",
		RedirectPort: 49152,
		CodeVerifier: "verifier",
		State:        "state",
		HTTPClient:   server.Client(),
		TokenURL:     server.URL + "/oauth/token",
		DashboardURL: server.URL,
	})
	if err == nil {
		t.Fatal("expected error for mismatched team credentials without --force")
	}
	var dashErr Error
	if !errors.As(err, &dashErr) || dashErr.Code != ErrorPlaidCredentialsOverwriteRequired {
		t.Fatalf("expected credential write error, got %v", err)
	}
	envContent, _ := os.ReadFile(envPath)
	if !strings.Contains(string(envContent), "PLAID_CLIENT_ID=team-a-client") {
		t.Fatalf("env should not be overwritten without force:\n%s", string(envContent))
	}
}

func TestRunLoginOverwritesMismatchedTeamCredentialsWithForce(t *testing.T) {
	configPath, envPath := writeLoginConfig(t, `
providers:
  plaid:
    client_id:
      env: PLAID_CLIENT_ID
    secret:
      env: PLAID_SECRET
    environment: sandbox
`)
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\nPLAID_CLIENT_ID=team-a-client\nPLAID_SECRET=team-a-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := loginFakeDashboard(t, "team-b-client", "team-b-secret")
	defer server.Close()
	result, err := RunLogin(context.Background(), LoginOptions{
		ConfigPath:   configPath,
		Environment:  "sandbox",
		Force:        true,
		CallbackCode: "auth-code",
		RedirectPort: 49152,
		CodeVerifier: "verifier",
		State:        "state",
		HTTPClient:   server.Client(),
		TokenURL:     server.URL + "/oauth/token",
		DashboardURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunLogin: %v", err)
	}
	if result.CredentialAction != "written" || result.KeysWritten != 2 {
		t.Fatalf("result = %#v", result)
	}
	envContent, _ := os.ReadFile(envPath)
	if !strings.Contains(string(envContent), "PLAID_CLIENT_ID=team-b-client") || !strings.Contains(string(envContent), "PLAID_SECRET=team-b-secret") {
		t.Fatalf("env should be overwritten with force:\n%s", string(envContent))
	}
}

func TestRunLoginPromptsForMultiTeamSelection(t *testing.T) {
	configPath, _ := writeLoginConfig(t, "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			w.Write([]byte(`{"access_token":"a","refresh_token":"r","expires_in":3600,"token_type":"Bearer"}`))
		case "/cli/teams/list":
			w.Write([]byte(`{"teams":[{"team_id":"team_1","client_id":"client_1","company":"First"},{"team_id":"team_2","client_id":"client_2","company":"Second"}]}`))
		case "/cli/keys/fetch":
			w.Write([]byte(`{"client_id":"client_2","secrets":{"sandbox":["sandbox-secret"]}}`))
		}
	}))
	defer server.Close()

	result, err := RunLogin(context.Background(), LoginOptions{
		ConfigPath:   configPath,
		Environment:  "sandbox",
		TeamPrompt:   prompt.NewFake("team_2"),
		CallbackCode: "auth-code",
		RedirectPort: 49152,
		CodeVerifier: "verifier",
		State:        "state",
		HTTPClient:   server.Client(),
		TokenURL:     server.URL + "/oauth/token",
		DashboardURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunLogin: %v", err)
	}
	if result.TeamID != "team_2" {
		t.Fatalf("result = %#v", result)
	}
	cfg, err := config.Load(config.Options{ConfigPath: configPath})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Providers["plaid"].Fields["client_id"] != "client_2" || cfg.Providers["plaid"].Fields["secret"] != "sandbox-secret" {
		t.Fatalf("fields = %#v", cfg.Providers["plaid"].Fields)
	}
}

func TestRunLoginWritesPlaidOptionsFromFlagsOrDefaults(t *testing.T) {
	configPath, _ := writeLoginConfig(t, "")
	server := loginFakeDashboard(t, "client_1", "sandbox-secret")
	defer server.Close()

	_, err := RunLogin(context.Background(), LoginOptions{
		ConfigPath:   configPath,
		Environment:  "sandbox",
		Products:     "transactions,liabilities",
		CountryCodes: "US,CA",
		RedirectURI:  "https://example.test/plaid/callback",
		CallbackCode: "auth-code",
		RedirectPort: 49152,
		CodeVerifier: "verifier",
		State:        "state",
		HTTPClient:   server.Client(),
		TokenURL:     server.URL + "/oauth/token",
		DashboardURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunLogin: %v", err)
	}
	cfg, err := config.Load(config.Options{ConfigPath: configPath})
	if err != nil {
		t.Fatal(err)
	}
	fields := cfg.Providers["plaid"].Fields
	if fields["products"] != "transactions,liabilities" || fields["country_codes"] != "US,CA" || fields["redirect_uri"] != "https://example.test/plaid/callback" {
		t.Fatalf("fields = %#v", fields)
	}

	defaultConfigPath, _ := writeLoginConfig(t, "")
	_, err = RunLogin(context.Background(), LoginOptions{
		ConfigPath:   defaultConfigPath,
		Environment:  "sandbox",
		CallbackCode: "auth-code",
		RedirectPort: 49152,
		CodeVerifier: "verifier",
		State:        "state",
		HTTPClient:   server.Client(),
		TokenURL:     server.URL + "/oauth/token",
		DashboardURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunLogin defaults: %v", err)
	}
	defaultCfg, err := config.Load(config.Options{ConfigPath: defaultConfigPath})
	if err != nil {
		t.Fatal(err)
	}
	defaultFields := defaultCfg.Providers["plaid"].Fields
	if defaultFields["products"] != "transactions" || defaultFields["country_codes"] != "US" || defaultFields["redirect_uri"] != "" {
		t.Fatalf("defaultFields = %#v", defaultFields)
	}
}

func TestRunLoginRequiresForceForEnvironmentSwitch(t *testing.T) {
	configPath, envPath := writeLoginConfig(t, `
providers:
  plaid:
    client_id:
      env: PLAID_CLIENT_ID
    secret:
      env: PLAID_SECRET
    environment: sandbox
`)
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\nPLAID_CLIENT_ID=existing-client\nPLAID_SECRET=existing-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := loginFakeDashboard(t, "fetched-client", "prod-secret")
	defer server.Close()
	_, err := RunLogin(context.Background(), LoginOptions{
		ConfigPath:   configPath,
		Environment:  "production",
		CallbackCode: "auth-code",
		RedirectPort: 49152,
		CodeVerifier: "verifier",
		State:        "state",
		HTTPClient:   server.Client(),
		TokenURL:     server.URL + "/oauth/token",
		DashboardURL: server.URL,
	})
	if !isPlaidLoginCode(err, ErrorPlaidCredentialsOverwriteRequired) {
		t.Fatalf("err = %#v", err)
	}
}

func TestRunLoginDoesNotWritePartialCredentialsWhenDashboardStepsFail(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"token": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "rejected", http.StatusUnauthorized)
		},
		"teams": func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/oauth/token":
				w.Write([]byte(`{"access_token":"a","refresh_token":"r","expires_in":3600,"token_type":"Bearer"}`))
			case "/cli/teams/list":
				http.Error(w, "no teams", http.StatusUnauthorized)
			}
		},
		"keys": func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/oauth/token":
				w.Write([]byte(`{"access_token":"a","refresh_token":"r","expires_in":3600,"token_type":"Bearer"}`))
			case "/cli/teams/list":
				w.Write([]byte(`{"teams":[{"team_id":"team_1","client_id":"client_1","company":"Acme"}]}`))
			case "/cli/keys/fetch":
				http.Error(w, "no keys", http.StatusUnauthorized)
			}
		},
		"missing-environment": func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/oauth/token":
				w.Write([]byte(`{"access_token":"a","refresh_token":"r","expires_in":3600,"token_type":"Bearer"}`))
			case "/cli/teams/list":
				w.Write([]byte(`{"teams":[{"team_id":"team_1","client_id":"client_1","company":"Acme"}]}`))
			case "/cli/keys/fetch":
				w.Write([]byte(`{"client_id":"client_1","secrets":{"production":["prod-secret"]}}`))
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			configPath, envPath := writeLoginConfig(t, "")
			server := httptest.NewServer(handler)
			defer server.Close()
			_, err := RunLogin(context.Background(), LoginOptions{
				ConfigPath:   configPath,
				Environment:  "sandbox",
				CallbackCode: "auth-code",
				RedirectPort: 49152,
				CodeVerifier: "verifier",
				State:        "state",
				HTTPClient:   server.Client(),
				TokenURL:     server.URL + "/oauth/token",
				DashboardURL: server.URL,
			})
			if err == nil {
				t.Fatal("expected error")
			}
			envContent, readErr := os.ReadFile(envPath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if strings.Contains(string(envContent), "PLAID_CLIENT_ID") || strings.Contains(string(envContent), "PLAID_SECRET") {
				t.Fatalf("partial credentials written:\n%s", string(envContent))
			}
			if _, readErr := ReadAuthFile(DashboardAuthPath(configPath)); readErr == nil {
				t.Fatal("auth file should not be written before complete key selection")
			}
		})
	}
}

func loginFakeDashboard(t *testing.T, clientID string, secret string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			w.Write([]byte(`{"access_token":"a","refresh_token":"r","expires_in":3600,"token_type":"Bearer"}`))
		case "/cli/teams/list":
			w.Write([]byte(`{"teams":[{"team_id":"team_1","client_id":"` + clientID + `","company":"Acme"}]}`))
		case "/cli/keys/fetch":
			w.Write([]byte(`{"client_id":"` + clientID + `","secrets":{"sandbox":["` + secret + `"],"production":["` + secret + `"]}}`))
		}
	}))
}

func writeLoginConfig(t *testing.T, providersYAML string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	envPath := filepath.Join(dir, ".env")
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if providersYAML == "" {
		providersYAML = "providers: {}\n"
	}
	if err := os.WriteFile(configPath, []byte(`
database:
  path: ./money.db
  encryption_key:
    env: MONEY_DB_ENCRYPTION_KEY
`+providersYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return configPath, envPath
}
