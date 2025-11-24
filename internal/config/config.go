package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/peaberberian/paul-envs/internal/utils"
)

type Shell string

const (
	VersionNone   = utils.VersionNone
	VersionLatest = utils.VersionLatest
)

const (
	ShellBash Shell = "bash"
	ShellZsh  Shell = "zsh"
	ShellFish Shell = "fish"
)

func (s *Shell) Set(value string) error {
	switch Shell(value) {
	case ShellBash, ShellZsh, ShellFish:
		*s = Shell(value)
		return nil
	default:
		return fmt.Errorf("invalid shell: %s", value)
	}
}

func (s Shell) String() string { return string(s) }

// Bool helper for tri-state
func Bool(v bool) *bool { return &v }

type Config struct {
	// The unique identifier, per-user, with which the corresponding image will
	// be identified
	ProjectName string

	// Name of the container's user, inside that container (not the host)
	Username string

	// Default shell linked to the user in the container
	Shell Shell

	// UID of the container's user.
	// It's better to synchronize it with the host to avoid permission issues when
	// host directories are mounted inside the container, such as the project
	// directory and dotfiles
	UID string

	// GID of the container's user.
	// It's better to synchronize it with the host to avoid permission issues when
	// host directories are mounted inside the container, such as the project
	// directory and dotfiles
	GID string

	// Whether to install the following languages and their tools, and the
	//version wanted.
	//
	// Values can be:
	// - if 'none': don't install Node.js
	// - if 'latest': Install Ubuntu's default package for Node.js
	// - If anything else: The exact version to install (e.g. "1.90.0").
	//   That last type of value will only work if INSTALL_MISE is 'true'.
	InstallNode   string
	InstallRust   string
	InstallPython string
	InstallGo     string

	// If 'true', add WebAssembly-specialized tools such as binaryen and a
	// WebAssembly target for Rust if it is installed.
	EnableWasm bool

	// If 'true', openssh will be installed
	EnableSsh bool

	// If 'true', sudo will be installed
	EnableSudo bool

	// Tools toggle.
	// "true" == install it
	// anything else == don't.
	InstallNeovim   bool
	InstallStarship bool
	InstallAtuin    bool
	InstallMise     bool
	InstallZellij   bool
	InstallJujutsu  bool

	Ports    []uint16
	Volumes  []string
	Packages []string

	// TODO: optionals?
	GitName         string
	GitEmail        string
	ProjectHostPath string
	ProjectDestPath string
	SshKeyPath      string
}

// New creates a config with UID/GID auto-detected.
func New(username string, shell Shell) Config {
	uid := "1000"
	gid := "1000"

	if runtime.GOOS != "windows" {
		uid = strconv.Itoa(os.Getuid())
		gid = strconv.Itoa(os.Getgid())
	}

	return Config{
		Username:      username,
		Shell:         shell,
		UID:           uid,
		GID:           gid,
		InstallNode:   "none",
		InstallRust:   "none",
		InstallPython: "none",
		InstallGo:     "none",
	}
}
