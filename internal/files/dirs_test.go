package files

import (
	"path/filepath"
	"testing"
)

// helper to restore detectOS after test
func mockOS(t *testing.T, osName string) {
	orig := detectOS
	detectOS = func() string { return osName }
	t.Cleanup(func() { detectOS = orig })
}

// helper to mock home lookup
func mockHome(t *testing.T, home string, err error) {
	orig := lookupHome
	lookupHome = func() (string, error) { return home, err }
	t.Cleanup(func() { lookupHome = orig })
}

//
// ─────────────────────────────────────────────────────────────
//   TEST getUserConfigDir()
// ─────────────────────────────────────────────────────────────
//

func TestGetUserConfigDir_Linux_XDG(t *testing.T) {
	mockOS(t, "linux")
	mockHome(t, "/home/test", nil)
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
	t.Setenv("SUDO_USER", "")

	dir, err := getUserConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dir != "/xdg/config/paul-envs" {
		t.Errorf("expected /xdg/config/paul-envs, got %s", dir)
	}
}

func TestGetUserConfigDir_Linux_NoXDG(t *testing.T) {
	mockOS(t, "linux")
	mockHome(t, "/home/test", nil)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("SUDO_USER", "")

	dir, err := getUserConfigDir()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	if dir != "/home/test/.config/paul-envs" {
		t.Errorf("expected /home/test/.config/paul-envs, got %s", dir)
	}
}

func TestGetUserConfigDir_Linux_Sudo(t *testing.T) {
	mockOS(t, "linux")
	mockHome(t, "/home/real", nil)
	t.Setenv("SUDO_USER", "real")
	t.Setenv("XDG_CONFIG_HOME", "/wrong/xdg")

	dir, err := getUserConfigDir()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	if dir != "/home/real/.config/paul-envs" {
		t.Errorf("expected /home/real/.config/paul-envs, got %s", dir)
	}
}

func TestGetUserConfigDir_MacOS(t *testing.T) {
	mockOS(t, "darwin")
	t.Setenv("HOME", "/Users/test")

	dir, err := getUserConfigDir()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	expected := "/Users/test/Library/Preferences/paul-envs"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestGetUserConfigDir_Windows(t *testing.T) {
	mockOS(t, "windows")
	t.Setenv("APPDATA", "C:\\Users\\test\\AppData\\Roaming")

	dir, err := getUserConfigDir()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	expected := filepath.Join("C:\\Users\\test\\AppData\\Roaming", "paul-envs")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

//
// ─────────────────────────────────────────────────────────────
//   TEST getUserDataDir()
// ─────────────────────────────────────────────────────────────
//

func TestGetUserDataDir_Linux_XDG(t *testing.T) {
	mockOS(t, "linux")
	mockHome(t, "/home/test", nil)
	t.Setenv("XDG_DATA_HOME", "/xdg/data")
	t.Setenv("SUDO_USER", "")

	dir, err := getUserDataDir()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	if dir != "/xdg/data/paul-envs" {
		t.Errorf("expected /xdg/data/paul-envs, got %s", dir)
	}
}

func TestGetUserDataDir_Linux_NoXDG(t *testing.T) {
	mockOS(t, "linux")
	mockHome(t, "/home/test", nil)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("SUDO_USER", "")

	dir, err := getUserDataDir()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	if dir != "/home/test/.local/share/paul-envs" {
		t.Errorf("expected /home/test/.local/share/paul-envs, got %s", dir)
	}
}

func TestGetUserDataDir_Linux_Sudo(t *testing.T) {
	mockOS(t, "linux")
	mockHome(t, "/home/real", nil)
	t.Setenv("SUDO_USER", "real")
	t.Setenv("XDG_DATA_HOME", "/wrong/xdg")

	dir, err := getUserDataDir()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	if dir != "/home/real/.local/share/paul-envs" {
		t.Errorf("expected /home/real/.local/share/paul-envs, got %s", dir)
	}
}

func TestGetUserDataDir_MacOS(t *testing.T) {
	mockOS(t, "darwin")
	t.Setenv("HOME", "/Users/test")

	dir, err := getUserDataDir()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	expected := "/Users/test/Library/Application Support/paul-envs"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestGetUserDataDir_Windows(t *testing.T) {
	mockOS(t, "windows")
	t.Setenv("APPDATA", "C:\\Users\\test\\AppData\\Roaming")

	dir, err := getUserDataDir()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	expected := filepath.Join("C:\\Users\\test\\AppData\\Roaming", "paul-envs")
	if dir != expected {
		t.Errorf("expected: %s, got %s", expected, dir)
	}
}

//
// ─────────────────────────────────────────────────────────────
//   TEST resolveRealUserHome()
// ─────────────────────────────────────────────────────────────
//

func TestResolveRealUserHome_Normal(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	t.Setenv("SUDO_USER", "")

	home, err := resolveRealUserHome()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if home != "/home/test" {
		t.Errorf("expected /home/test, got %s", home)
	}
}
