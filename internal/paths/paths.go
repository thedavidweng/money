// Package paths resolves the data directory for the money CLI.
//
// The canonical data directory is determined in this order:
//  1. MONEY_HOME environment variable (explicit override)
//  2. Legacy ~/.money directory (backward compatibility if it exists)
//  3. Platform-appropriate default derived from os.UserHomeDir()
//
// On Linux this respects XDG_STATE_HOME (default ~/.local/state/money).
// On macOS it uses ~/Library/Application Support/money.
// On Windows it uses %LOCALAPPDATA%\money.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

const appName = "money"

// DataDir returns the root directory for all money data files
// (config.yaml, .env, data/money.db, plaid-dashboard-auth.json, profiles/).
func DataDir() string {
	if override := os.Getenv("MONEY_HOME"); override != "" {
		return override
	}

	home, _ := os.UserHomeDir()

	// Prefer legacy ~/.money if it exists and the new platform path does not.
	if home != "" {
		legacy := filepath.Join(home, ".money")
		if _, err := os.Stat(legacy); err == nil {
			newDefault := platformDefault(home)
			if _, err := os.Stat(newDefault); err != nil {
				return legacy
			}
		}
	}

	return platformDefault(home)
}

// platformDefault returns the XDG/platform-appropriate data directory.
// Returns "" if home is empty and no platform-specific env var is set,
// since a CWD-relative path would silently scatter data across the filesystem.
func platformDefault(home string) string {
	switch runtime.GOOS {
	case "darwin":
		if home == "" {
			return ""
		}
		return filepath.Join(home, "Library", "Application Support", appName)
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, appName)
		}
		if home == "" {
			return ""
		}
		return filepath.Join(home, "AppData", "Local", appName)
	default: // linux, freebsd, etc.
		if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
			return filepath.Join(xdg, appName)
		}
		if home == "" {
			return ""
		}
		return filepath.Join(home, ".local", "state", appName)
	}
}
