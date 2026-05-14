package plaidlogin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestExchangeCodeUsesPKCEFormBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		assertForm(t, r.PostForm, "grant_type", "authorization_code")
		assertForm(t, r.PostForm, "code", "auth-code")
		assertForm(t, r.PostForm, "redirect_uri", "http://localhost:49152/oauth/callback")
		assertForm(t, r.PostForm, "client_id", ClientID)
		assertForm(t, r.PostForm, "code_verifier", "verifier")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"expires_in":    3600,
			"token_type":    "Bearer",
			"scope":         "openid",
		})
	}))
	defer server.Close()

	auth, err := ExchangeCode(context.Background(), TokenClientConfig{
		TokenURL:   server.URL,
		HTTPClient: server.Client(),
		Now:        func() time.Time { return time.Unix(100, 0).UTC() },
	}, ExchangeCodeRequest{Code: "auth-code", RedirectURI: RedirectURI(49152), CodeVerifier: "verifier"})
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if auth.AccessToken != "access-token" || auth.RefreshToken != "refresh-token" {
		t.Fatalf("auth = %#v", auth)
	}
	if !auth.ExpiresAt.Equal(time.Unix(3700, 0).UTC()) {
		t.Fatalf("ExpiresAt = %s", auth.ExpiresAt)
	}
}

func TestRefreshTokenClassifiesMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"new"}`))
	}))
	defer server.Close()

	_, err := RefreshToken(context.Background(), TokenClientConfig{TokenURL: server.URL, HTTPClient: server.Client()}, "refresh-token")
	var dashErr Error
	if !errors.As(err, &dashErr) || dashErr.Code != ErrorDashboardContractChanged {
		t.Fatalf("err = %#v", err)
	}
}

func assertForm(t *testing.T, form url.Values, key, want string) {
	t.Helper()
	if got := form.Get(key); got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}
