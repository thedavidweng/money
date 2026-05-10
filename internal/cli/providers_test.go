package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProvidersPlaidLinkJSONMissingCredentialsReturnsStableAuthError(t *testing.T) {
	configPath := writeTestConfig(t, "")

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"--config", configPath, "providers", "plaid", "link", "--json", "--no-open"}, nil, &stdout, &stderr)

	if exitCode != 3 {
		t.Fatalf("exit code = %d, want 3; stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty JSON-mode diagnostics", stderr.String())
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Errors []struct {
			Code     string `json:"code"`
			Category string `json:"category"`
		} `json:"errors"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if envelope.OK {
		t.Fatal("ok = true, want false")
	}
	if envelope.Meta.Command != "providers.plaid.link" {
		t.Fatalf("command = %q", envelope.Meta.Command)
	}
	if len(envelope.Errors) != 1 || envelope.Errors[0].Code != "PROVIDER_CREDENTIALS_MISSING" || envelope.Errors[0].Category != "auth" {
		t.Fatalf("errors = %#v", envelope.Errors)
	}
}

func TestProvidersBridgeLinkJSONMissingCredentialsReturnsStableAuthError(t *testing.T) {
	configPath := writeTestConfig(t, "")

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"--config", configPath, "providers", "bridge", "link", "--json", "--no-open"}, nil, &stdout, &stderr)

	if exitCode != 3 {
		t.Fatalf("exit code = %d, want 3; stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty JSON-mode diagnostics", stderr.String())
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Errors []struct {
			Code     string `json:"code"`
			Category string `json:"category"`
		} `json:"errors"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if envelope.OK {
		t.Fatal("ok = true, want false")
	}
	if envelope.Meta.Command != "providers.bridge.link" {
		t.Fatalf("command = %q", envelope.Meta.Command)
	}
	if len(envelope.Errors) != 1 || envelope.Errors[0].Code != "PROVIDER_CREDENTIALS_MISSING" || envelope.Errors[0].Category != "auth" {
		t.Fatalf("errors = %#v", envelope.Errors)
	}
}

func TestInstitutionLinkJSONMissingCredentialsReportsUnavailableProvider(t *testing.T) {
	configPath := writeTestConfig(t, "")

	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"--config", configPath, "link", "bank", "--json"}, nil, &stdout, &stderr)

	if exitCode != 3 {
		t.Fatalf("exit code = %d, want 3; stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty JSON-mode diagnostics", stderr.String())
	}

	var envelope struct {
		OK     bool `json:"ok"`
		Errors []struct {
			Code     string `json:"code"`
			Category string `json:"category"`
			Message  string `json:"message"`
		} `json:"errors"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if envelope.OK {
		t.Fatal("ok = true, want false")
	}
	if envelope.Meta.Command != "link" {
		t.Fatalf("command = %q", envelope.Meta.Command)
	}
	if len(envelope.Errors) != 1 || envelope.Errors[0].Code != "PROVIDER_CREDENTIALS_MISSING" || envelope.Errors[0].Category != "auth" {
		t.Fatalf("errors = %#v", envelope.Errors)
	}
	if !strings.Contains(envelope.Errors[0].Message, "unavailable locally") {
		t.Fatalf("message = %q", envelope.Errors[0].Message)
	}
}

func writeTestConfig(t *testing.T, providers string) string {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	envPath := filepath.Join(dir, ".env")
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if providers == "" {
		providers = "providers: {}\n"
	}
	if err := os.WriteFile(configPath, []byte(`
database:
  path: ./money.db
  encryption_key:
    env: MONEY_DB_ENCRYPTION_KEY
`+providers), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return configPath
}
