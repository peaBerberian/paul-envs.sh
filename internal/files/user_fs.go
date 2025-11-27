package files

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
)

// Isolated for tests
var detectOS = func() string {
	return runtime.GOOS
}

type UserFS struct {
	homeDir  string
	sudoUser *sudoUser
}

type sudoUser struct {
	uid int
	gid int
}

func NewUserFS() (*UserFS, error) {
	sudoUserEnv := os.Getenv("SUDO_USER")
	if sudoUserEnv == "" {
		var home string
		if detectOS() == "windows" {
			home = os.Getenv("USERPROFILE")
		} else {
			home = os.Getenv("HOME")
		}
		if home == "" {
			return nil, errors.New("cannot determine HOME: HOME not set")
		}
		return &UserFS{sudoUser: nil, homeDir: home}, nil
	}
	usr, err := user.Lookup(sudoUserEnv)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve info on the SUDO_USER: %w", err)
	}
	uid, _ := strconv.Atoi(usr.Uid)
	gid, _ := strconv.Atoi(usr.Gid)
	return &UserFS{
		sudoUser: &sudoUser{
			uid: uid,
			gid: gid,
		},
		homeDir: usr.HomeDir,
	}, nil
}

func (u *UserFS) MkdirAsUser(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return err
	}
	if u.sudoUser != nil {
		return os.Chown(path, u.sudoUser.uid, u.sudoUser.gid)
	}
	return nil
}

func (u *UserFS) GetUserDataDir() string {
	switch detectOS() {
	case "windows":
		appDataEnv := os.Getenv("APPDATA")
		if appDataEnv == "" {
			return filepath.Join(u.homeDir, "AppData", "Local")
		} else {
			// TODO: Conflict with config
			return appDataEnv
		}
	case "darwin":
		return filepath.Join(u.homeDir, "Library", "Application Support")
	default: // linux / unix
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" && u.sudoUser == nil {
			return xdg
		} else {
			// sudo don't preserve the original XDG_DATA_HOME
			// So we just use the default XDG location
			return filepath.Join(u.homeDir, ".local", "share")
		}
	}
}

func (u *UserFS) GetUserConfigDir() string {
	switch detectOS() {
	case "windows":
		appDataEnv := os.Getenv("APPDATA")
		if appDataEnv == "" {
			return filepath.Join(u.homeDir, "AppData", "Roaming")
		}
		return appDataEnv

	case "darwin":
		return filepath.Join(u.homeDir, "Library", "Preferences")

	default: // Linux / Unix
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" && u.sudoUser == nil {
			return xdg
		} else {
			// sudo don't preserve the original XDG_CONFIG_HOME
			// So we just use the default XDG location
			return filepath.Join(u.homeDir, ".config")
		}
	}
}

// CopyDir recursively copies a directory tree from src to dst
// TODO: Pass context
func (u *UserFS) CopyDirAsUser(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return u.MkdirAsUser(target, info.Mode())
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		// TODO: as user
		dstFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
