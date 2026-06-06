package plaidlogin

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func DashboardAuthPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "plaid-dashboard-auth.json")
}

func ReadAuthFile(path string) (Auth, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Auth{}, Error{Code: ErrorNotLoggedIn, Message: "Plaid Dashboard auth is not present"}
		}
		return Auth{}, err
	}
	var auth Auth
	if err := json.Unmarshal(content, &auth); err != nil {
		return Auth{}, Error{Code: ErrorDashboardContractChanged, Message: "Plaid Dashboard auth file is malformed", Err: err}
	}
	if auth.AccessToken == "" || auth.RefreshToken == "" || auth.ExpiresAt.IsZero() {
		return Auth{}, Error{Code: ErrorDashboardContractChanged, Message: "Plaid Dashboard auth file omitted required fields"}
	}
	return auth, nil
}

func WriteAuthFile(path string, auth Auth) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	content, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".plaid-dashboard-auth-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(append(content, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpName, 0o600); err != nil {
			return err
		}
	}
	return os.Rename(tmpName, path)
}

func DeleteAuthFile(path string) (bool, error) {
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func LoadFreshAuth(ctx context.Context, path string, cfg TokenClientConfig) (Auth, error) {
	auth, err := ReadAuthFile(path)
	if err != nil {
		return Auth{}, err
	}
	now := cfg.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if auth.ExpiresAt.After(now()) {
		return auth, nil
	}
	refreshed, err := RefreshToken(ctx, cfg, auth.RefreshToken)
	if err != nil {
		return Auth{}, Error{Code: ErrorDashboardTokenRefreshFailed, Message: "Stored Plaid Dashboard auth could not be refreshed", Err: err}
	}
	refreshed.TeamID = auth.TeamID
	refreshed.ClientID = auth.ClientID
	if err := WriteAuthFile(path, refreshed); err != nil {
		return Auth{}, err
	}
	return refreshed, nil
}
