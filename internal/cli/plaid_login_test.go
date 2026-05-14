package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/thedavidweng/money/internal/plaidlogin"
)

func TestPlaidLoginJSONMissingBaseConfigReturnsStableError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing.yaml")
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"--config", configPath, "--json", "plaid", "login", "--no-open"}, nil, &stdout, &stderr)
	if exitCode != 3 {
		t.Fatalf("exit code = %d stdout=%s stderr=%s", exitCode, stdout.String(), stderr.String())
	}
	var envelope struct {
		OK     bool `json:"ok"`
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if envelope.OK || envelope.Meta.Command != "plaid.login" || len(envelope.Errors) != 1 || envelope.Errors[0].Code != "BASE_CONFIG_MISSING" {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestPlaidLogoutDeletesOnlyDashboardAuth(t *testing.T) {
	configPath, envPath := writePlaidLoginTestConfig(t)
	authPath := filepath.Join(filepath.Dir(configPath), "plaid-dashboard-auth.json")
	if err := os.WriteFile(authPath, []byte(`{"access_token":"a","refresh_token":"r","expires_at":"2026-05-13T00:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"--config", configPath, "--json", "plaid", "logout"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d stdout=%s stderr=%s", exitCode, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(authPath); !os.IsNotExist(err) {
		t.Fatalf("auth file still exists or stat failed: %v", err)
	}
	envContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(envContent), "PLAID_SECRET=secret") {
		t.Fatalf("provider env was changed:\n%s", string(envContent))
	}
	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			DashboardAuthRemoved bool `json:"dashboard_auth_removed"`
			APIKeysPreserved     bool `json:"api_keys_preserved"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if !envelope.OK || !envelope.Data.DashboardAuthRemoved || !envelope.Data.APIKeysPreserved {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestProvidersPlaidLoginAliasExists(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"providers", "plaid", "login", "--help"}, nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d stderr=%s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--environment") {
		t.Fatalf("help did not include login flags: %s", stderr.String())
	}
}

func TestPlaidLoginCommandsUseSharedFakeAndPreserveStderr(t *testing.T) {
	oldRunPlaidLogin := runPlaidLoginCLI
	t.Cleanup(func() { runPlaidLoginCLI = oldRunPlaidLogin })

	var commands []string
	runPlaidLoginCLI = func(ctx context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer, opts plaidLoginCLIOptions) error {
		commands = append(commands, opts.CommandName)
		if _, err := fmt.Fprintln(stderr, "oauth progress"); err != nil {
			return err
		}
		return writePlaidLoginResult(state, stdout, plaidlogin.LoginResult{
			Provider:          "plaid",
			TeamID:            "team_1",
			Environment:       opts.Environment,
			KeysWritten:       2,
			CredentialAction:  "written",
			DashboardAuthPath: "/tmp/plaid-dashboard-auth.json",
			NextCommand:       "money link <institution-query>",
			ConfigPath:        "/tmp/config.yaml",
			EnvPath:           "/tmp/.env",
		}, opts.CommandName)
	}

	for _, args := range [][]string{
		{"--json", "plaid", "login", "--no-open"},
		{"--json", "providers", "plaid", "login", "--no-open"},
	} {
		var stdout, stderr bytes.Buffer
		exitCode := Run(context.Background(), args, nil, &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("%v exit code = %d stdout=%s stderr=%s", args, exitCode, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "oauth progress") {
			t.Fatalf("%v dropped stderr: %q", args, stderr.String())
		}
		var envelope struct {
			OK   bool `json:"ok"`
			Meta struct {
				Command string `json:"command"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
			t.Fatalf("%v stdout is not JSON: %v\n%s", args, err, stdout.String())
		}
		if !envelope.OK {
			t.Fatalf("%v envelope = %#v", args, envelope)
		}
	}
	if strings.Join(commands, ",") != "plaid.login,providers.plaid.login" {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestPlaidLoginJSONRequiresForceBeforeOAuthForEnvironmentSwitch(t *testing.T) {
	configPath, _ := writePlaidLoginTestConfig(t)
	var stdout, stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := runPlaidLoginLive(ctx, &runtimeState{
		configPath: configPath,
		profile:    "default",
		json:       true,
	}, &stdout, &stderr, plaidLoginCLIOptions{
		CommandName: "plaid.login",
		NoOpen:      true,
		Environment: "production",
	})
	cliErr, ok := err.(cliError)
	if !ok {
		t.Fatalf("err = %#v", err)
	}
	if cliErr.code != "CONFIRMATION_REQUIRED" || cliErr.exitCode != 10 {
		t.Fatalf("cliErr = %#v", cliErr)
	}
	if strings.Contains(stderr.String(), "OAuth URL") {
		t.Fatalf("OAuth started before overwrite validation: %s", stderr.String())
	}
}

func TestPlaidLoginAndLogoutRespectReadOnlyBeforeOAuthOrFileWrites(t *testing.T) {
	configPath, _ := writePlaidLoginTestConfig(t)
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, append([]byte("read_only: true\n"), content...), 0o600); err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(filepath.Dir(configPath), "plaid-dashboard-auth.json")
	if err := os.WriteFile(authPath, []byte(`{"access_token":"a","refresh_token":"r","expires_at":"2026-05-13T00:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var loginStdout, loginStderr bytes.Buffer
	loginCode := Run(context.Background(), []string{"--config", configPath, "--json", "plaid", "login", "--no-open"}, nil, &loginStdout, &loginStderr)
	if loginCode != 4 {
		t.Fatalf("login exit code = %d stdout=%s stderr=%s", loginCode, loginStdout.String(), loginStderr.String())
	}
	if strings.Contains(loginStderr.String(), "OAuth URL") {
		t.Fatalf("OAuth started in read-only mode: %s", loginStderr.String())
	}

	var logoutStdout, logoutStderr bytes.Buffer
	logoutCode := Run(context.Background(), []string{"--config", configPath, "--json", "plaid", "logout"}, nil, &logoutStdout, &logoutStderr)
	if logoutCode != 4 {
		t.Fatalf("logout exit code = %d stdout=%s stderr=%s", logoutCode, logoutStdout.String(), logoutStderr.String())
	}
	if _, err := os.Stat(authPath); err != nil {
		t.Fatalf("auth file should remain in read-only mode: %v", err)
	}
}

func TestDoctorWarnsOnBroadPlaidDashboardAuthPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode warnings are skipped on Windows")
	}
	configPath, _ := writePlaidLoginTestConfig(t)
	authPath := filepath.Join(filepath.Dir(configPath), "plaid-dashboard-auth.json")
	if err := os.WriteFile(authPath, []byte(`{"access_token":"a","refresh_token":"r","expires_at":"2026-05-13T00:00:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	diagnostics := runDiagnostics(configPath, "default")
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "PLAID_DASHBOARD_AUTH_PERMISSIONS" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing dashboard auth permission warning: %#v", diagnostics)
	}
}

func writePlaidLoginTestConfig(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	envPath := filepath.Join(dir, ".env")
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if err := os.WriteFile(configPath, []byte(`
database:
  path: ./money.db
  encryption_key:
    env: MONEY_DB_ENCRYPTION_KEY
providers:
  plaid:
    client_id:
      env: PLAID_CLIENT_ID
    secret:
      env: PLAID_SECRET
    environment: sandbox
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\nPLAID_CLIENT_ID=client\nPLAID_SECRET=secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return configPath, envPath
}
