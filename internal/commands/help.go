package commands

import (
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func Help(filestore *files.FileStore, console *console.Console) {
	console.WriteLn(`paul-envs - Development Environment Manager

Usage:
  paul-envs create <path> [options]
  paul-envs list
  paul-envs build <name>
  paul-envs run <name> [commands]
  paul-envs remove <name>
  paul-envs version
  paul-envs help
  paul-envs interactive
  paul-envs clean

Options for create (all optional):
  --no-prompt              Non-interactive mode (uses defaults)
  --name NAME              Name of this project (default: directory name)
  --uid UID                Container UID (default: current user - or 1000 on windows)
  --gid GID                Container GID (default: current group - or 1000 on windows)
  --username NAME          Container username (default: dev)
  --shell SHELL            User shell: bash|zsh|fish (prompted if not specified)
  --nodejs VERSION         Node.js installation:
                             'none' - skip installation of Node.js
                             'latest' - use Ubuntu default package
                             '20.10.0' - specific version (requires mise)
                           (prompted if no language specified)
  --rust VERSION           Rust installation:
                             'none' - skip installation of Rust
                             'latest' - latest stable via rustup
                             '1.75.0' - specific version (requires mise)
                           (prompted if no language specified)
  --python VERSION         Python installation:
                             'none' - skip installation of Python
                             'latest' - use Ubuntu default package
                             '3.12.0' - specific version (requires mise)
                           (prompted if no language specified)
  --go VERSION             Go installation:
                             'none' - skip installation of Go
                             'latest' - use Ubuntu default package
                             '1.21.5' - specific version (requires mise)
                           (prompted if no language specified)
  --enable-wasm            Add WASM-specialized tools (binaryen, Rust wasm target if enabled)
                           (prompted if no language specified)
  --enable-ssh             Enable ssh access on port 22 (E.g. to access files from your host)
                           (prompted if not specified)
  --enable-sudo            Enable sudo access in container with a "dev" password
                           (prompted if not specified)
  --git-name NAME          Git user.name (optional)
  --git-email EMAIL        Git user.email (optional)
  --neovim                 Install Neovim (text editor)
                           (prompted if no tool specified)
  --starship               Install Starship (prompt)
                           (prompted if no tool specified)
  --atuin                  Install Atuin (shell history)
                           (prompted if no tool specified)
  --mise                   Install Mise (version manager - required for specific language versions)
                           (prompted if no tool specified)
  --zellij                 Install Zellij (terminal multiplexer)
                           (prompted if no tool specified)
  --jujutsu                Install Jujutsu (Git-compatible VCS)
                           (prompted if no tool specified)
  --package PKG_NAME       Additional Ubuntu package (prompted if not specified, can be repeated)
  --port PORT              Expose container port (prompted if not specified, can be repeated)
  --volume HOST:CONT[:ro]  Mount volume (prompted if not specified, can be repeated)

Windows/Git Bash Notes:
  - UID/GID default to 1000 on Windows (Docker Desktop requirement)

Creating a configuration in interactive Mode (default):
  paul-envs create ~/projects/myapp
  # Will prompt for all unspecified options

Creating a configuration in a non-Interactive Mode:
  paul-envs create ~/projects/myapp --no-prompt --shell bash --nodejs latest

Mixed Mode (some flags + prompts):
  paul-envs create ~/projects/myapp --nodejs 20.10.0 --rust latest --mise
  # Will prompt for shell, sudo, packages, ports, and volumes

Full Configuration Example:
  paul-envs create ~/work/api \
    --name myApp \
    --shell zsh \
    --nodejs 20.10.0 \
    --rust latest \
    --python 3.12.0 \
    --go latest \
    --mise \
    --neovim \
    --starship \
    --zellij \
    --jujutsu \
    --enable-ssh \
    --enable-sudo \
    --git-name "John Doe" \
    --git-email "john@example.com" \
    --package ripgrep \
    --package ripgrep \
    --port 3000 \
    --port 5432 \
    --volume ~/.git-credentials:/home/dev/.git-credentials:ro

Location of stored files created by this tool:
  Base compose       : ` + filestore.GetBaseComposeFile() + `
  Projects directory : ` + filestore.GetProjectDirBase() + `

NOTE: To start a guided prompt, you can also just run:
  paul-envs interactive
`)
}
