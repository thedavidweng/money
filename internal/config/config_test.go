package config

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResolvesExplicitEnvReferencesFromCompanionEnv(t *testing.T) {
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
    products: [transactions]
    country_codes: [US]
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\nPLAID_CLIENT_ID=client\nPLAID_SECRET=secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(Options{ConfigPath: configPath, Env: map[string]string{}})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DatabasePath != filepath.Join(dir, "money.db") {
		t.Fatalf("database path = %q", cfg.DatabasePath)
	}
	if cfg.DatabaseEncryptionKey != key {
		t.Fatalf("database key was not resolved from explicit env reference")
	}
	if got := string(cfg.DatabaseEncryptionKeyBytes); got != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("database key bytes = %q", got)
	}
	if cfg.Providers["plaid"].Fields["client_id"] != "client" {
		t.Fatalf("plaid client_id was not resolved")
	}
}

func TestLoadDoesNotReadCwdEnvWithoutExplicitConfigReference(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("MONEY_DB_ENCRYPTION_KEY=from_cwd\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = Load(Options{ConfigPath: filepath.Join(dir, "missing.yaml"), Env: map[string]string{}})
	if err == nil {
		t.Fatal("load succeeded from cwd .env, want missing config error")
	}
}

func TestLoadResolvesBridgeExplicitEnvReferences(t *testing.T) {
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
  bridge:
    client_id:
      env: BRIDGE_CLIENT_ID
    client_secret:
      env: BRIDGE_CLIENT_SECRET
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\nBRIDGE_CLIENT_ID=bridge-client\nBRIDGE_CLIENT_SECRET=bridge-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(Options{ConfigPath: configPath, Env: map[string]string{}})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Providers["bridge"].Fields["client_id"] != "bridge-client" {
		t.Fatal("BRIDGE_CLIENT_ID was not resolved")
	}
	if cfg.Providers["bridge"].Fields["client_secret"] != "bridge-secret" {
		t.Fatal("BRIDGE_CLIENT_SECRET was not resolved")
	}
}
