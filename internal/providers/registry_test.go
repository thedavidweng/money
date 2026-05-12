package providers

import (
	"context"
	"testing"

	"github.com/thedavidweng/money/internal/config"
)

func TestRegistryIncludesPlaidAndBridge(t *testing.T) {
	registry := NewRegistry(config.Config{Providers: map[string]config.ProviderConfig{}})

	if _, ok := registry.Get("plaid"); !ok {
		t.Fatal("plaid provider missing from registry")
	}
	if _, ok := registry.Get("bridge"); !ok {
		t.Fatal("bridge provider missing from registry")
	}
}

func TestMissingProviderCredentialsProduceStableConfigDiagnostics(t *testing.T) {
	registry := NewRegistry(config.Config{Providers: map[string]config.ProviderConfig{}})

	plaid, _ := registry.Get("plaid")
	plaidDiagnostics := plaid.ValidateConfig(context.Background())
	if len(plaidDiagnostics) != 1 || plaidDiagnostics[0].Code != "PROVIDER_CREDENTIALS_MISSING" {
		t.Fatalf("plaid diagnostics = %#v", plaidDiagnostics)
	}

	bridge, _ := registry.Get("bridge")
	bridgeDiagnostics := bridge.ValidateConfig(context.Background())
	if len(bridgeDiagnostics) != 1 || bridgeDiagnostics[0].Code != "PROVIDER_CREDENTIALS_MISSING" {
		t.Fatalf("bridge diagnostics = %#v", bridgeDiagnostics)
	}
}
