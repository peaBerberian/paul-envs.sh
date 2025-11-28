package files

import (
	"os"
	"path/filepath"
	"testing"
)

// helper to restore detectOS after test
func mockOS(t *testing.T, osName string) {
	orig := detectOS
	detectOS = func() string { return osName }
	t.Cleanup(func() { detectOS = orig })
}

//
// ─────────────────────────────────────────────────────────────
//   TEST GetUserConfigDir()
// ─────────────────────────────────────────────────────────────
//

func TestGetUserConfigDir_Linux_XDG(t *testing.T) {
	mockOS(t, "linux")
	t.Setenv("HOME", "/home/test")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dir := ufs.GetUserConfigDir()
	if dir != "/xdg/config" {
		t.Errorf("expected /xdg/config, got %s", dir)
	}
}

func TestGetUserConfigDir_Linux_NoXDG(t *testing.T) {
	mockOS(t, "linux")
	t.Setenv("HOME", "/home/test")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	dir := ufs.GetUserConfigDir()
	if dir != "/home/test/.config" {
		t.Errorf("expected /home/test/.config, got %s", dir)
	}
}

func TestGetUserConfigDir_Linux_Sudo(t *testing.T) {
	mockOS(t, "linux")
	t.Setenv("HOME", "/root")
	t.Setenv("SUDO_USER", os.Getenv("USER")) // Use actual current user for lookup
	t.Setenv("XDG_CONFIG_HOME", "/wrong/xdg")

	ufs, err := NewUserFS()
	if err != nil {
		t.Skipf("skipping sudo test: %v", err)
	}

	dir := ufs.GetUserConfigDir()
	// Should use real user's home, not XDG_CONFIG_HOME
	if filepath.Base(dir) != ".config" {
		t.Errorf("expected path ending in .config, got %s", dir)
	}
}

func TestGetUserConfigDir_MacOS(t *testing.T) {
	mockOS(t, "darwin")
	t.Setenv("HOME", "/Users/test")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	dir := ufs.GetUserConfigDir()
	expected := "/Users/test/Library/Preferences"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestGetUserConfigDir_Windows(t *testing.T) {
	mockOS(t, "windows")
	t.Setenv("USERPROFILE", "C:\\Users\\test")
	t.Setenv("APPDATA", "C:\\Users\\test\\AppData\\Roaming")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	dir := ufs.GetUserConfigDir()
	expected := "C:\\Users\\test\\AppData\\Roaming"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestGetUserConfigDir_Windows_NoAppData(t *testing.T) {
	mockOS(t, "windows")
	t.Setenv("USERPROFILE", "C:\\Users\\test")
	t.Setenv("APPDATA", "")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	dir := ufs.GetUserConfigDir()
	expected := filepath.Join("C:\\Users\\test", "AppData", "Roaming")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

//
// ─────────────────────────────────────────────────────────────
//   TEST GetUserDataDir()
// ─────────────────────────────────────────────────────────────
//

func TestGetUserDataDir_Linux_XDG(t *testing.T) {
	mockOS(t, "linux")
	t.Setenv("HOME", "/home/test")
	t.Setenv("XDG_DATA_HOME", "/xdg/data")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	dir := ufs.GetUserDataDir()
	if dir != "/xdg/data" {
		t.Errorf("expected /xdg/data, got %s", dir)
	}
}

func TestGetUserDataDir_Linux_NoXDG(t *testing.T) {
	mockOS(t, "linux")
	t.Setenv("HOME", "/home/test")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	dir := ufs.GetUserDataDir()
	if dir != "/home/test/.local/share" {
		t.Errorf("expected /home/test/.local/share, got %s", dir)
	}
}

func TestGetUserDataDir_Linux_Sudo(t *testing.T) {
	mockOS(t, "linux")
	t.Setenv("HOME", "/root")
	t.Setenv("SUDO_USER", os.Getenv("USER"))
	t.Setenv("XDG_DATA_HOME", "/wrong/xdg")

	ufs, err := NewUserFS()
	if err != nil {
		t.Skipf("skipping sudo test: %v", err)
	}

	dir := ufs.GetUserDataDir()
	// Should use real user's home, not XDG_DATA_HOME
	if filepath.Base(dir) != "share" {
		t.Errorf("expected path ending in .local/share, got %s", dir)
	}
}

func TestGetUserDataDir_MacOS(t *testing.T) {
	mockOS(t, "darwin")
	t.Setenv("HOME", "/Users/test")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	dir := ufs.GetUserDataDir()
	expected := "/Users/test/Library/Application Support"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestGetUserDataDir_Windows(t *testing.T) {
	mockOS(t, "windows")
	t.Setenv("USERPROFILE", "C:\\Users\\test")
	t.Setenv("LOCALAPPDATA", "C:\\Users\\test\\AppData\\Local")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	dir := ufs.GetUserDataDir()
	expected := "C:\\Users\\test\\AppData\\Local"
	if dir != expected {
		t.Errorf("expected: %s, got %s", expected, dir)
	}
}

func TestGetUserDataDir_Windows_NoLocalAppData(t *testing.T) {
	mockOS(t, "windows")
	t.Setenv("USERPROFILE", "C:\\Users\\test")
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	dir := ufs.GetUserDataDir()
	expected := filepath.Join("C:\\Users\\test", "AppData", "Local")
	if dir != expected {
		t.Errorf("expected: %s, got %s", expected, dir)
	}
}

//
// ─────────────────────────────────────────────────────────────
//   TEST NewUserFS()
// ─────────────────────────────────────────────────────────────
//

func TestNewUserFS_Normal(t *testing.T) {
	mockOS(t, "linux")
	t.Setenv("HOME", "/home/test")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if ufs.homeDir != "/home/test" {
		t.Errorf("expected /home/test, got %s", ufs.homeDir)
	}
	if ufs.sudoUser != nil {
		t.Errorf("expected no sudo user, got %+v", ufs.sudoUser)
	}
}

func TestNewUserFS_Windows(t *testing.T) {
	mockOS(t, "windows")
	t.Setenv("USERPROFILE", "C:\\Users\\test")
	t.Setenv("HOME", "")
	t.Setenv("SUDO_USER", "")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if ufs.homeDir != "C:\\Users\\test" {
		t.Errorf("expected C:\\Users\\test, got %s", ufs.homeDir)
	}
}

func TestNewUserFS_NoHome(t *testing.T) {
	mockOS(t, "linux")
	t.Setenv("HOME", "")
	t.Setenv("SUDO_USER", "")

	_, err := NewUserFS()
	if err == nil {
		t.Fatal("expected error when HOME not set")
	}
}
