package config

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigSkeletonWithHomePrefix(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	dbPath := filepath.Join(home, "data", "money.db")
	skel := configSkeleton(dbPath)

	if !strings.Contains(skel, "~/data/money.db") {
		t.Fatalf("configSkeleton(%q) missing ~-prefixed path:\n%s", dbPath, skel)
	}
	if strings.Contains(skel, home) {
		t.Fatalf("configSkeleton(%q) leaked absolute home path:\n%s", dbPath, skel)
	}
	if !strings.Contains(skel, "encryption_key:") {
		t.Fatalf("configSkeleton missing encryption_key:\n%s", skel)
	}
	if !strings.Contains(skel, "providers: {}") {
		t.Fatalf("configSkeleton missing providers:\n%s", skel)
	}
}

func TestConfigSkeletonWithoutHomePrefix(t *testing.T) {
	dbPath := "/tmp/test-project/money.db"
	skel := configSkeleton(dbPath)

	if !strings.Contains(skel, dbPath) {
		t.Fatalf("configSkeleton(%q) missing absolute path:\n%s", dbPath, skel)
	}
	if strings.Contains(skel, "~") {
		t.Fatalf("configSkeleton(%q) incorrectly used ~ prefix:\n%s", dbPath, skel)
	}
}

func TestProviderSpecByNamePlaid(t *testing.T) {
	spec, ok := ProviderSpecByName("plaid")
	if !ok {
		t.Fatal("ProviderSpecByName(\"plaid\") returned false")
	}
	if spec.Name != "plaid" {
		t.Fatalf("name = %q, want plaid", spec.Name)
	}
	if len(spec.SecretFields) == 0 {
		t.Fatal("plaid secret fields is empty")
	}
	if spec.HelpURL == "" {
		t.Fatal("plaid help URL is empty")
	}
}

func TestProviderSpecByNameBridge(t *testing.T) {
	spec, ok := ProviderSpecByName("bridge")
	if !ok {
		t.Fatal("ProviderSpecByName(\"bridge\") returned false")
	}
	if spec.Name != "bridge" {
		t.Fatalf("name = %q, want bridge", spec.Name)
	}
	if len(spec.SecretFields) == 0 {
		t.Fatal("bridge secret fields is empty")
	}
}

func TestProviderSpecByNameUnknown(t *testing.T) {
	_, ok := ProviderSpecByName("unknown_provider")
	if ok {
		t.Fatal("ProviderSpecByName(\"unknown_provider\") returned true, want false")
	}
}

func TestEnsureEncryptionKeyCreatesEnvFile(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	created, err := ensureEncryptionKey(envPath, false)
	if err != nil {
		t.Fatalf("ensureEncryptionKey: %v", err)
	}
	if !created {
		t.Fatal("ensureEncryptionKey returned false, want true on first call")
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if !strings.Contains(string(content), "MONEY_DB_ENCRYPTION_KEY=") {
		t.Fatalf(".env missing MONEY_DB_ENCRYPTION_KEY:\n%s", string(content))
	}

	// Extract and validate the key.
	lines := strings.Split(string(content), "\n")
	var keyLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "MONEY_DB_ENCRYPTION_KEY=") {
			keyLine = strings.TrimPrefix(line, "MONEY_DB_ENCRYPTION_KEY=")
			break
		}
	}
	if keyLine == "" {
		t.Fatal("MONEY_DB_ENCRYPTION_KEY value is empty")
	}
	keyBytes, err := base64.RawURLEncoding.DecodeString(keyLine)
	if err != nil {
		t.Fatalf("key is not valid base64url: %v", err)
	}
	if len(keyBytes) != 32 {
		t.Fatalf("key decodes to %d bytes, want 32", len(keyBytes))
	}
}

func TestEnsureEncryptionKeyDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	// First call creates the key.
	if _, err := ensureEncryptionKey(envPath, false); err != nil {
		t.Fatalf("first ensureEncryptionKey: %v", err)
	}
	content1, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env first time: %v", err)
	}

	// Second call without force should not overwrite.
	created, err := ensureEncryptionKey(envPath, false)
	if err != nil {
		t.Fatalf("second ensureEncryptionKey: %v", err)
	}
	if created {
		t.Fatal("second ensureEncryptionKey returned true, want false (key already exists)")
	}
	content2, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env second time: %v", err)
	}
	if string(content1) != string(content2) {
		t.Fatalf(".env changed between calls without force:\nfirst:\n%s\nsecond:\n%s", content1, content2)
	}
}

func TestEnsureEncryptionKeyForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	// First call creates the key.
	if _, err := ensureEncryptionKey(envPath, false); err != nil {
		t.Fatalf("first ensureEncryptionKey: %v", err)
	}
	content1, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env first time: %v", err)
	}

	// Force overwrite should create a new key.
	created, err := ensureEncryptionKey(envPath, true)
	if err != nil {
		t.Fatalf("force ensureEncryptionKey: %v", err)
	}
	if !created {
		t.Fatal("force ensureEncryptionKey returned false, want true")
	}
	content2, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env second time: %v", err)
	}
	if string(content1) == string(content2) {
		t.Fatal(".env not changed after force overwrite")
	}
}

func TestEnsureEncryptionKeyPreservesOtherVars(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	// Write a .env with other variables.
	if err := os.WriteFile(envPath, []byte("PLAID_CLIENT_ID=abc\nOTHER_VAR=hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	created, err := ensureEncryptionKey(envPath, false)
	if err != nil {
		t.Fatalf("ensureEncryptionKey: %v", err)
	}
	if !created {
		t.Fatal("ensureEncryptionKey returned false, want true")
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "PLAID_CLIENT_ID=abc") {
		t.Fatalf(".env lost PLAID_CLIENT_ID:\n%s", s)
	}
	if !strings.Contains(s, "OTHER_VAR=hello") {
		t.Fatalf(".env lost OTHER_VAR:\n%s", s)
	}
	if !strings.Contains(s, "MONEY_DB_ENCRYPTION_KEY=") {
		t.Fatalf(".env missing MONEY_DB_ENCRYPTION_KEY:\n%s", s)
	}
}

func TestSetupCreatesConfigDirAndFiles(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	result, err := Setup(configPath, "", false)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if result.ConfigPath != configPath {
		t.Fatalf("config path = %q, want %q", result.ConfigPath, configPath)
	}
	if !result.SecretCreated {
		t.Fatal("secret_created = false, want true on first run")
	}

	// Verify config.yaml was created.
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config.yaml was not created")
	}

	// Verify .env was created with encryption key.
	envPath := filepath.Join(dir, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Fatal(".env was not created")
	}
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if !strings.Contains(string(content), "MONEY_DB_ENCRYPTION_KEY=") {
		t.Fatalf(".env missing key:\n%s", string(content))
	}

	// Verify config.yaml references the key.
	cfgContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config.yaml: %v", err)
	}
	if !strings.Contains(string(cfgContent), "MONEY_DB_ENCRYPTION_KEY") {
		t.Fatalf("config.yaml missing env reference:\n%s", string(cfgContent))
	}
}

func TestSetupDoesNotOverwriteExistingConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	// Pre-create config.yaml with custom content.
	customContent := "database:\n  path: /custom/path.db\n"
	if err := os.WriteFile(configPath, []byte(customContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Setup(configPath, "", false)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if result.ConfigPath != configPath {
		t.Fatalf("config path = %q, want %q", result.ConfigPath, configPath)
	}

	// Config should be preserved.
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config.yaml: %v", err)
	}
	if string(content) != customContent {
		t.Fatalf("config.yaml was overwritten:\ngot:\n%s\nwant:\n%s", string(content), customContent)
	}
}

func TestSetupRejectsInvalidProfileWithExplicitPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	_, err := Setup(configPath, "../../etc", false)
	if err == nil {
		t.Fatal("Setup with invalid profile should error")
	}
}
