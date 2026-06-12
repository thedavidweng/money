package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDataDirHonorsMoneyHomeOverride(t *testing.T) {
	t.Setenv("MONEY_HOME", "/custom/money/path")
	got := DataDir()
	if got != "/custom/money/path" {
		t.Fatalf("DataDir() = %q, want /custom/money/path", got)
	}
}

func TestDataDirUsesNewPlatformPathWhenLegacyMissing(t *testing.T) {
	// Clean environment
	t.Setenv("MONEY_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// No ~/.money exists → use new platform default
	got := DataDir()
	want := platformDefault(home)
	if got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

func TestDataDirFallsBackToLegacyWhenExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MONEY_HOME", "")

	legacyDir := filepath.Join(home, ".money")
	if err := os.MkdirAll(legacyDir, 0o700); err != nil {
		t.Fatal(err)
	}

	got := DataDir()
	if got != legacyDir {
		t.Fatalf("DataDir() = %q, want legacy %q", got, legacyDir)
	}
}

func TestDataDirPrefersNewPathWhenBothExist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MONEY_HOME", "")

	legacyDir := filepath.Join(home, ".money")
	newDir := platformDefault(home)
	for _, d := range []string{legacyDir, newDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	got := DataDir()
	if got != newDir {
		t.Fatalf("DataDir() = %q, want new path %q (both exist, prefer new)", got, newDir)
	}
}

func TestDataDirDefaultLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MONEY_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")

	got := DataDir()
	want := filepath.Join(home, ".local", "state", "money")
	if got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

func TestDataDirLinuxXDGStateHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only")
	}
	xdgDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MONEY_HOME", "")

	got := DataDir()
	want := filepath.Join(xdgDir, "money")
	if got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

func TestDataDirDefaultWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only")
	}
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("MONEY_HOME", "")

	got := DataDir()
	want := filepath.Join(localAppData, "money")
	if got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

func TestDataDirDefaultDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MONEY_HOME", "")

	got := DataDir()
	want := filepath.Join(home, "Library", "Application Support", "money")
	if got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

func TestDataDirReturnsEmptyWhenHomeUnavailable(t *testing.T) {
	t.Setenv("MONEY_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	if runtime.GOOS == "linux" {
		t.Setenv("XDG_STATE_HOME", "")
	}
	if runtime.GOOS == "windows" {
		t.Setenv("LOCALAPPDATA", "")
	}

	got := DataDir()
	if got != "" {
		t.Fatalf("DataDir() = %q, want empty string when home dir is unavailable", got)
	}
}
