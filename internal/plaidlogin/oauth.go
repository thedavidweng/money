package plaidlogin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
)

const (
	AuthURL      = "https://dashboard.plaid.com/oauth/authorize"
	TokenURL     = "https://api.dashboard.plaid.com/oauth/token"
	ClientID     = "plaid-cli"
	BindHost     = "127.0.0.1"
	RedirectHost = "localhost"

	// Plaid Dashboard APIs are private; this contract was last verified against this Plaid CLI build.
	PlaidCLICompatibilityVersion = "20260507-4d1b0ca0"
)

type AuthConfig struct {
	Port         int
	State        string
	CodeVerifier string
}

func NewRandomString(byteCount int) (string, error) {
	bytes := make([]byte, byteCount)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func BuildAuthURL(cfg AuthConfig) (string, error) {
	if cfg.Port <= 0 {
		return "", fmt.Errorf("OAuth callback port is required")
	}
	if cfg.State == "" {
		return "", fmt.Errorf("OAuth state is required")
	}
	if cfg.CodeVerifier == "" {
		return "", fmt.Errorf("OAuth PKCE verifier is required")
	}
	values := url.Values{}
	values.Set("client_id", ClientID)
	values.Set("redirect_uri", RedirectURI(cfg.Port))
	values.Set("response_type", "code")
	values.Set("state", cfg.State)
	values.Set("code_challenge_method", "S256")
	values.Set("code_challenge", pkceChallenge(cfg.CodeVerifier))
	return AuthURL + "?" + values.Encode(), nil
}

func RedirectURI(port int) string {
	return fmt.Sprintf("http://%s:%d/oauth/callback", RedirectHost, port)
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
