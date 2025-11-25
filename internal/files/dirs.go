package files

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

// Isolated for tests
var detectOS = func() string {
	return runtime.GOOS
}

// Isolated for tests
var lookupHome = resolveRealUserHome

func resolveRealUserHome() (string, error) {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" && sudoUser != "root" {
		u, err := user.Lookup(sudoUser)
		if err != nil {
			return "", fmt.Errorf("failed to lookup sudo user %s: %w", sudoUser, err)
		}
		return u.HomeDir, nil
	}

	// Not running under sudo or actual root user
	home := os.Getenv("HOME")
	if home == "" {
		return "", errors.New("cannot determine HOME: HOME not set")
	}
	return home, nil
}

func getUserDataDir() (string, error) {
	var baseDataDir string

	switch detectOS() {
	case "windows":
		baseDataDir = os.Getenv("APPDATA")
		if baseDataDir == "" {
			baseDataDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
	case "darwin":
		baseDataDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	default: // linux / unix
		homeDir, err := lookupHome()
		if err != nil {
			return "", err
		}
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" && os.Getenv("SUDO_USER") == "" {
			baseDataDir = xdg
		} else {
			// sudo don't preserve the original XDG_DATA_HOME
			// So we just use the default XDG location
			baseDataDir = filepath.Join(homeDir, ".local", "share")
		}
	}
	return filepath.Join(baseDataDir, "paul-envs"), nil
}

func getUserConfigDir() (string, error) {
	var baseDataDir string

	switch detectOS() {
	case "windows":
		// Windows: %APPDATA%
		baseDataDir = os.Getenv("APPDATA")
		if baseDataDir == "" {
			baseDataDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}

	case "darwin":
		home := os.Getenv("HOME")
		if home == "" {
			return "", errors.New("cannot determine HOME on macOS")
		}
		baseDataDir = filepath.Join(home, "Library", "Preferences")

	default: // Linux / Unix
		homeDir, err := lookupHome()
		if err != nil {
			return "", err
		}
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" && os.Getenv("SUDO_USER") == "" {
			baseDataDir = xdg
		} else {
			// sudo don't preserve the original XDG_CONFIG_HOME
			// So we just use the default XDG location
			baseDataDir = filepath.Join(homeDir, ".config")
		}
	}
	return filepath.Join(baseDataDir, "paul-envs"), nil
}
