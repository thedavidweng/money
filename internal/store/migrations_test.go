package store

import (
	"os"
	"strings"
	"testing"
)

func TestInitialMigrationMatchesSchemaDocument(t *testing.T) {
	migration, err := os.ReadFile("migrations/0001_initial.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	schema, err := os.ReadFile("../../docs/SCHEMA.md")
	if err != nil {
		t.Fatalf("read schema doc: %v", err)
	}

	migrationSQL := strings.TrimSpace(string(migration))
	if !strings.Contains(string(schema), migrationSQL) {
		t.Fatal("docs/SCHEMA.md DDL must contain the exact 0001_initial.sql migration text")
	}
}
