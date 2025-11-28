package files

import (
	"os/user"
	"path/filepath"
	"testing"
)

// helper to restore detectOS after test
func mockOS(t *testing.T, osName string) {
	orig := detectOS
	detectOS = func() string { return osName }
	t.Cleanup(func() { detectOS = orig })
}

func mockGeteuid(t *testing.T, uid int) {
	orig := geteuid
	geteuid = func() int { return uid }
	t.Cleanup(func() { geteuid = orig })
}

func mockUserLookup(t *testing.T, fn func(string) (*user.User, error)) {
	orig := userLookup
	userLookup = fn
	t.Cleanup(func() { userLookup = orig })
}

//
// ─────────────────────────────────────────────────────────────
//   TEST GetUserConfigDir()
// ─────────────────────────────────────────────────────────────
//

func TestGetUserConfigDir_Linux_XDG(t *testing.T) {
	mockOS(t, "linux")
	mockGeteuid(t, 1000)
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
	mockGeteuid(t, 1000)
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
	mockGeteuid(t, 0) // Running as root
	mockUserLookup(t, func(username string) (*user.User, error) {
		return &user.User{
			Uid:     "1000",
			Gid:     "1000",
			HomeDir: "/home/testuser",
		}, nil
	})
	t.Setenv("HOME", "/root")
	t.Setenv("SUDO_USER", "testuser")
	t.Setenv("XDG_CONFIG_HOME", "/wrong/xdg")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dir := ufs.GetUserConfigDir()
	expected := "/home/testuser/.config"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestGetUserConfigDir_MacOS(t *testing.T) {
	mockOS(t, "darwin")
	mockGeteuid(t, 501)
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
	mockGeteuid(t, 1000)
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
	mockGeteuid(t, 1000)
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
	mockGeteuid(t, 1000)
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
	mockGeteuid(t, 1000)
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
	mockGeteuid(t, 0)
	mockUserLookup(t, func(username string) (*user.User, error) {
		return &user.User{
			Uid:     "1000",
			Gid:     "1000",
			HomeDir: "/home/testuser",
		}, nil
	})
	t.Setenv("HOME", "/root")
	t.Setenv("SUDO_USER", "testuser")
	t.Setenv("XDG_DATA_HOME", "/wrong/xdg")

	ufs, err := NewUserFS()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dir := ufs.GetUserDataDir()
	expected := "/home/testuser/.local/share"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestGetUserDataDir_MacOS(t *testing.T) {
	mockOS(t, "darwin")
	mockGeteuid(t, 501)
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
	mockGeteuid(t, 1000)
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
	mockGeteuid(t, 1000)
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
	mockGeteuid(t, 1000)
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
	mockGeteuid(t, 1000)
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
	mockGeteuid(t, 1000)
	t.Setenv("HOME", "")
	t.Setenv("SUDO_USER", "")

	_, err := NewUserFS()
	if err == nil {
		t.Fatal("expected error when HOME not set")
	}
}
