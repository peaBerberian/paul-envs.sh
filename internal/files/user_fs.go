package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
)

// Isolated for tests
var geteuid = os.Geteuid
var userLookup = user.Lookup
var detectOS = func() string {
	return runtime.GOOS
}

// Allows to perform filesystem operations setting the current user as the
// owner. The current user may be running sudo to run as root, in which
// case we try to set permission as the sudo's user
type UserFS struct {
	// The home directory where the user's files reside
	homeDir string
	// Only set if it has been detected that the user is running under `sudo`
	sudoUser *sudoUser
}

// Information linked to a user running sudo
type sudoUser struct {
	// The `uid` of the user running sudo
	uid int
	// The `gid` of the user running sudo
	gid int
}

// Create a new `UserFS`, allowing to perform operations on the filesystem as
// the current user (who may be running sudo).
func NewUserFS() (*UserFS, error) {
	sudoUserEnv := os.Getenv("SUDO_USER")
	if geteuid() != 0 || sudoUserEnv == "" {
		var home string
		if detectOS() == "windows" {
			home = os.Getenv("USERPROFILE")
		} else {
			home = os.Getenv("HOME")
		}
		if home == "" {
			return nil, errors.New("cannot determine the home directory, your system might not be supported")
		}
		return &UserFS{sudoUser: nil, homeDir: home}, nil
	}
	usr, err := userLookup(sudoUserEnv)
	if err != nil {
		return nil, fmt.Errorf("running sudo but cannot retrieve info on the SUDO_USER: %w", err)
	}
	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		return nil, fmt.Errorf("failed to convert the sudo user uid into an integer: %w", err)
	}
	gid, err := strconv.Atoi(usr.Gid)
	if err != nil {
		return nil, fmt.Errorf("failed to convert the sudo user gid into an integer: %w", err)
	}
	return &UserFS{
		sudoUser: &sudoUser{
			uid: uid,
			gid: gid,
		},
		homeDir: usr.HomeDir,
	}, nil
}

// Create a directory with the associated file permissions and the set the
// current user as the owner
func (u *UserFS) MkdirAsUser(path string, perm os.FileMode) error {
	existed := true
	if _, err := os.Stat(path); os.IsNotExist(err) {
		existed = false
	}
	if err := os.MkdirAll(path, perm); err != nil {
		return err
	}
	if err := u.chownIfNeeded(path); err != nil {
		if !existed {
			// NOTE: this is not perfect as this does not clean-up the potential
			// parent directories that have been created under `MkdirAll`...
			// Maybe a TODO for later.
			os.RemoveAll(path)
		}
		return err
	}
	return nil
}

// Create a file with the associated file permissions and the set the
// current user as the owner
func (u *UserFS) WriteFileAsUser(path string, data []byte, perm os.FileMode) error {
	existed := true
	if _, err := os.Stat(path); os.IsNotExist(err) {
		existed = false
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		return err
	}
	if err := u.chownIfNeeded(path); err != nil {
		if !existed {
			os.Remove(path)
		}
		return err
	}
	return nil
}

// Returns the "data" directory associated with this user, where application
// data can reside.
func (u *UserFS) GetUserDataDir() string {
	switch detectOS() {
	case "windows":
		localAppDataEnv := os.Getenv("LOCALAPPDATA")
		if localAppDataEnv == "" {
			return filepath.Join(u.homeDir, "AppData", "Local")
		} else {
			return localAppDataEnv
		}
	case "darwin":
		return filepath.Join(u.homeDir, "Library", "Application Support")
	default: // linux / unix
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" && u.sudoUser == nil {
			return xdg
		} else {
			// sudo doesn't preserve the original XDG_DATA_HOME
			// So we just use the default XDG location
			return filepath.Join(u.homeDir, ".local", "share")
		}
	}
}

// Returns the "config" directory associated with this user, where application
// configuration, that the user can update, can reside.
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

// Recursively copies a directory tree from src to dst and set the user as the
// owner of dst files
func (u *UserFS) CopyDirAsUser(ctx context.Context, src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %q: %w", path, err)
		}

		// Context check
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("failed to obtain relative path from '%q' to '%q': %w", src, path, err)
		}
		target := filepath.Join(dst, rel)

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to get information on '%q': %w", path, err)
		}

		if d.IsDir() {
			if err := u.MkdirAsUser(target, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory '%q': %w", target, err)
			}
			return nil
		}

		// Handle symlink (copy symlink, not contents)
		if d.Type()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read the symlink '%q': %w", path, err)
			}
			if err := os.Symlink(linkTarget, target); err != nil {
				return fmt.Errorf("failed to create symlink '%q' to '%q': %w", linkTarget, target, err)
			}
			return u.chownIfNeeded(target)
		}

		if err := copyFileContext(ctx, path, target, info.Mode()); err != nil {
			return err
		}

		return u.chownIfNeeded(target)
	})
}

func copyFileContext(ctx context.Context, path, target string, mode fs.FileMode) error {
	in, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open '%q': %w", path, err)
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create or update '%q': %w", target, err)
	}
	defer out.Close()

	buf := make([]byte, 128*1024)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, rerr := in.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return fmt.Errorf("failed to write '%q': %w", target, werr)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fmt.Errorf("failed to read '%q': %w", path, rerr)
		}
	}
	return nil
}

func (u *UserFS) chownIfNeeded(path string) error {
	if u.sudoUser == nil || runtime.GOOS == "windows" {
		return nil
	}
	// NOTE: `Lchown` is used, so a symlink itself has its owner changed, not the target
	if err := os.Lchown(path, u.sudoUser.uid, u.sudoUser.gid); err != nil {
		return fmt.Errorf("failed to change owner of '%q': %w", path, err)
	}
	return nil
}
