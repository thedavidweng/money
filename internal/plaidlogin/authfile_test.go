package plaidlogin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestAuthFileRoundTripAndDelete(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "profile", "config.yaml")
	authPath := DashboardAuthPath(configPath)
	wantPath := filepath.Join(dir, "profile", "plaid-dashboard-auth.json")
	if authPath != wantPath {
		t.Fatalf("DashboardAuthPath = %q, want %q", authPath, wantPath)
	}

	auth := Auth{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Unix(1778700000, 0).UTC(),
		TeamID:       "team_1",
		ClientID:     "client_1",
	}
	if err := WriteAuthFile(authPath, auth); err != nil {
		t.Fatalf("WriteAuthFile: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(authPath)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("perm = %o, want 600", perm)
		}
	}
	got, err := ReadAuthFile(authPath)
	if err != nil {
		t.Fatalf("ReadAuthFile: %v", err)
	}
	if got.AccessToken != auth.AccessToken || got.RefreshToken != auth.RefreshToken || got.TeamID != auth.TeamID || got.ClientID != auth.ClientID {
		t.Fatalf("auth = %#v", got)
	}
	removed, err := DeleteAuthFile(authPath)
	if err != nil {
		t.Fatalf("DeleteAuthFile: %v", err)
	}
	if !removed {
		t.Fatal("removed = false, want true")
	}
	removed, err = DeleteAuthFile(authPath)
	if err != nil {
		t.Fatalf("DeleteAuthFile missing: %v", err)
	}
	if removed {
		t.Fatal("removed missing = true, want false")
	}
}

func TestLoadFreshAuthRefreshesExpiredAuthFile(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "plaid-dashboard-auth.json")
	if err := WriteAuthFile(authPath, Auth{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Unix(100, 0).UTC(),
		TeamID:       "team_1",
		ClientID:     "client_1",
	}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.PostForm.Get("grant_type") != "refresh_token" || r.PostForm.Get("refresh_token") != "old-refresh" {
			t.Fatalf("refresh form = %#v", r.PostForm)
		}
		w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	auth, err := LoadFreshAuth(context.Background(), authPath, TokenClientConfig{
		TokenURL:   server.URL,
		HTTPClient: server.Client(),
		Now:        func() time.Time { return time.Unix(200, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("LoadFreshAuth: %v", err)
	}
	if auth.AccessToken != "new-access" || auth.RefreshToken != "new-refresh" || auth.TeamID != "team_1" || auth.ClientID != "client_1" {
		t.Fatalf("auth = %#v", auth)
	}
	stored, err := ReadAuthFile(authPath)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AccessToken != "new-access" || stored.TeamID != "team_1" || stored.ClientID != "client_1" {
		t.Fatalf("stored = %#v", stored)
	}
}

func TestLoadFreshAuthClassifiesRefreshFailure(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "plaid-dashboard-auth.json")
	if err := WriteAuthFile(authPath, Auth{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Unix(100, 0).UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "expired", http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := LoadFreshAuth(context.Background(), authPath, TokenClientConfig{
		TokenURL:   server.URL,
		HTTPClient: server.Client(),
		Now:        func() time.Time { return time.Unix(200, 0).UTC() },
	})
	var dashErr Error
	if !errors.As(err, &dashErr) || dashErr.Code != ErrorDashboardTokenRefreshFailed {
		t.Fatalf("err = %#v", err)
	}
}
