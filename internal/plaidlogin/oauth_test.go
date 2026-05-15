package plaidlogin

import (
	"net/url"
	"testing"
)

func TestBuildAuthURLUsesPlaidCLICompatiblePKCE(t *testing.T) {
	authURL, err := BuildAuthURL(AuthConfig{
		Port:         49152,
		State:        "state-123",
		CodeVerifier: "verifier-123",
	})
	if err != nil {
		t.Fatalf("BuildAuthURL: %v", err)
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path; got != "https://dashboard.plaid.com/oauth/authorize" {
		t.Fatalf("auth endpoint = %q", got)
	}
	values := parsed.Query()
	assertQuery(t, values, "client_id", "plaid-cli")
	assertQuery(t, values, "redirect_uri", "http://127.0.0.1:49152/oauth/callback")
	assertQuery(t, values, "response_type", "code")
	assertQuery(t, values, "state", "state-123")
	assertQuery(t, values, "code_challenge_method", "S256")
	if values.Get("code_challenge") == "" || values.Get("code_challenge") == "verifier-123" {
		t.Fatalf("code_challenge should be S256 hash, got %q", values.Get("code_challenge"))
	}
}

func TestBindAndRedirectHostsMatch(t *testing.T) {
	if BindHost != "127.0.0.1" {
		t.Fatalf("BindHost = %q", BindHost)
	}
	if RedirectHost != "127.0.0.1" {
		t.Fatalf("RedirectHost = %q", RedirectHost)
	}
}

func assertQuery(t *testing.T, values url.Values, key, want string) {
	t.Helper()
	if got := values.Get(key); got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}
