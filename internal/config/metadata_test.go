package config

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveMetadataHonorsExplicitEnvFileWithoutResolvingProviderSecrets(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	envPath := filepath.Join(dir, "secrets", "provider.env")
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

	if err := os.WriteFile(configPath, []byte(`
env_file: secrets/provider.env
read_only: true
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
	if err := os.MkdirAll(filepath.Dir(envPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	meta, err := ResolveMetadata(Options{ConfigPath: configPath, Env: map[string]string{}})
	if err != nil {
		t.Fatalf("ResolveMetadata: %v", err)
	}
	if meta.ConfigPath != configPath {
		t.Fatalf("ConfigPath = %q, want %q", meta.ConfigPath, configPath)
	}
	if meta.EnvPath != envPath {
		t.Fatalf("EnvPath = %q, want %q", meta.EnvPath, envPath)
	}
	if !meta.ReadOnly {
		t.Fatal("ReadOnly = false, want true")
	}
	if meta.DatabasePath != filepath.Join(dir, "money.db") {
		t.Fatalf("DatabasePath = %q", meta.DatabasePath)
	}
}

func TestResolveMetadataAndLoadHonorDotenvReadOnly(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	envPath := filepath.Join(dir, ".env")
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

	if err := os.WriteFile(configPath, []byte(`
database:
  path: ./money.db
  encryption_key:
    env: MONEY_DB_ENCRYPTION_KEY
providers: {}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\nMONEY_READ_ONLY=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	meta, err := ResolveMetadata(Options{ConfigPath: configPath, Env: map[string]string{}})
	if err != nil {
		t.Fatalf("ResolveMetadata: %v", err)
	}
	if !meta.ReadOnly {
		t.Fatal("metadata ReadOnly = false, want true")
	}
	cfg, err := Load(Options{ConfigPath: configPath, Env: map[string]string{}})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.ReadOnly {
		t.Fatal("config ReadOnly = false, want true")
	}
}

func TestConfigureProviderWritesResolvedEnvFileAndRefusesSilentSkip(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	envPath := filepath.Join(dir, "private", "secrets.env")
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

	if err := os.MkdirAll(filepath.Dir(envPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`
env_file: private/secrets.env
database:
  path: ./money.db
  encryption_key:
    env: MONEY_DB_ENCRYPTION_KEY
providers: {}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\nPLAID_CLIENT_ID=old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ConfigureProvider(configPath, "default", PlaidSpec, map[string]string{
		"client_id": "new-client",
		"secret":    "new-secret",
	}, map[string]string{"environment": "sandbox"}, false)
	if err == nil {
		t.Fatal("ConfigureProvider should reject replacing an existing env value without force")
	}

	result, err := ConfigureProvider(configPath, "default", PlaidSpec, map[string]string{
		"client_id": "new-client",
		"secret":    "new-secret",
	}, map[string]string{"environment": "sandbox"}, true)
	if err != nil {
		t.Fatalf("ConfigureProvider force: %v", err)
	}
	if result.EnvPath != envPath {
		t.Fatalf("EnvPath = %q, want %q", result.EnvPath, envPath)
	}
	if result.KeysWritten != 2 {
		t.Fatalf("KeysWritten = %d, want 2", result.KeysWritten)
	}
	envContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(envContent); !containsAll(got, "PLAID_CLIENT_ID=new-client", "PLAID_SECRET=new-secret") {
		t.Fatalf("env file did not contain forced credentials:\n%s", got)
	}
	cfg, err := Load(Options{ConfigPath: configPath, Env: map[string]string{}})
	if err != nil {
		t.Fatalf("Load after configure: %v", err)
	}
	if cfg.Providers["plaid"].Fields["client_id"] != "new-client" {
		t.Fatalf("client id = %q", cfg.Providers["plaid"].Fields["client_id"])
	}
}

func containsAll(value string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
