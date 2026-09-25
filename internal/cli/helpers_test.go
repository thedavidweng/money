package cli

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func writeTestConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	envPath := filepath.Join(dir, ".env")
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	if err := os.WriteFile(configPath, []byte(`
database:
  path: ./money.db
  encryption_key: "env:MONEY_DB_ENCRYPTION_KEY"
providers: {}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("MONEY_DB_ENCRYPTION_KEY="+key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return configPath
}
