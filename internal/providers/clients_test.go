package providers

import (
	"context"
	"net/http"
	"testing"
)

func TestPlaidClientUsesOfficialSDKAndDirectEnvironment(t *testing.T) {
	client, err := NewPlaidClient(PlaidClientConfig{
		ClientID:    "client-id",
		Secret:      "secret",
		Environment: "sandbox",
	})
	if err != nil {
		t.Fatalf("new plaid client: %v", err)
	}

	serverURL, err := client.Configuration.ServerURL(0, nil)
	if err != nil {
		t.Fatalf("server url: %v", err)
	}
	if serverURL != "https://sandbox.plaid.com" {
		t.Fatalf("server URL = %q, want Plaid sandbox", serverURL)
	}
	if client.Configuration.DefaultHeader["PLAID-CLIENT-ID"] != "client-id" {
		t.Fatal("PLAID-CLIENT-ID header was not set")
	}
	if client.Configuration.DefaultHeader["PLAID-SECRET"] != "secret" {
		t.Fatal("PLAID-SECRET header was not set")
	}
}

func TestPlaidClientRejectsUnknownEnvironmentInsteadOfFallingBack(t *testing.T) {
	_, err := NewPlaidClient(PlaidClientConfig{
		ClientID:    "client-id",
		Secret:      "secret",
		Environment: "ray-proxy",
	})
	if err == nil {
		t.Fatal("expected unknown Plaid environment error")
	}
}

func TestBridgeClientUsesDirectBridgeAPIBaseURL(t *testing.T) {
	client, err := NewBridgeClient(BridgeClientConfig{
		ClientID:     "client-id",
		ClientSecret: "secret",
	})
	if err != nil {
		t.Fatalf("new bridge client: %v", err)
	}

	if client.BaseURL != "https://api.bridgeapi.io/v3" {
		t.Fatalf("base URL = %q, want direct Bridge API", client.BaseURL)
	}
	if client.HTTPClient == nil || client.HTTPClient == http.DefaultClient {
		t.Fatal("Bridge client should own an HTTP client")
	}
}

func TestBridgeClientBuildsVersionedAuthenticatedRequest(t *testing.T) {
	client, err := NewBridgeClient(BridgeClientConfig{
		ClientID:     "client-id",
		ClientSecret: "secret",
	})
	if err != nil {
		t.Fatalf("new bridge client: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/aggregation/transactions", "access-token", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	if req.URL.String() != "https://api.bridgeapi.io/v3/aggregation/transactions" {
		t.Fatalf("url = %s", req.URL.String())
	}
	if req.Header.Get("Bridge-Version") != "2025-01-15" {
		t.Fatalf("Bridge-Version = %q", req.Header.Get("Bridge-Version"))
	}
	if req.Header.Get("Client-Id") != "client-id" || req.Header.Get("Client-Secret") != "secret" {
		t.Fatalf("Bridge credentials headers missing: %#v", req.Header)
	}
	if req.Header.Get("Authorization") != "Bearer access-token" {
		t.Fatalf("Authorization = %q", req.Header.Get("Authorization"))
	}
}
