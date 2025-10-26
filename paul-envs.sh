#!/bin/bash
set -e

# Small dance to resolve symlinks of this script (macOS compatible)
SCRIPT_PATH="${BASH_SOURCE[0]}"

# Follow symlink until the actual script location
while [ -L "$SCRIPT_PATH" ]; do
    # Directory path when cd-ing into where the symlink is
    SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
    # macOS readlink doesn't support -f, so we use plain readlink and do tricks
    # with it
    SCRIPT_PATH="$(readlink "$SCRIPT_PATH")"
    # Handle relative symlinks by doing a concatenation if doesn't start with
    # `/`
    [[ "$SCRIPT_PATH" != /* ]] && SCRIPT_PATH="$SCRIPT_DIR/$SCRIPT_PATH"
done
SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
BASE_COMPOSE="$SCRIPT_DIR/compose.yaml"

# Directory where projects' yaml and env files will be created
PROJECTS_DIR="$SCRIPT_DIR/projects"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
error() {
    printf "${RED}Error: %s${NC}\n" "$1" >&2;
    exit 1;
}
success() {
    printf "${GREEN}%s${NC}\n" "$1";
}
warn() {
    printf "${YELLOW}%s${NC}\n" "$1";
}
info() {
    printf "${BLUE}%s${NC}\n" "$1";
}

# Security validation functions
validate_project_name() {
    local name=$1
    # Alphanumeric + `-` + _ only
    if [[ ! "$name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        error "Invalid project name '$name'. Only alphanumeric characters, hyphens, and underscores are allowed."
    fi
}

# Check that the choosen shell is one supported by paul-envs.sh
# TODO: support nushell and others?
validate_shell() {
    local shell=$1
    case "$shell" in
        bash|zsh|fish)
            return 0
            ;;
        *)
            error "Invalid shell '$shell'. Must be one of: bash, zsh, fish"
            ;;
    esac
}

validate_version_arg() {
    local version=$1
    # Allow "none", latest" or semantic versioning patterns
    if [[ -n "$version" && "$version" != "latest" && "$version" != "none" && ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        error "Invalid version argument: '$version'. Must be either \"none\", \"latest\" or semantic versioning (e.g., 20.10.0)"
    fi
}

# Validate the `--packages` flag, which are supplementary Ubuntu packages to
# install
validate_apt_package_names() {
    local packages=$1
    # Each package name should match apt's allowed characters
    # https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-source
    for pkg in $packages; do
        if [[ ! "$pkg" =~ ^[a-z0-9+._-]+$ ]]; then
            error "Invalid package name '$pkg'. Package names must contain only lowercase letters, digits, hyphens, periods, and plus signs."
        fi
    done
}

# Sanitize string for YAML/env - escape quotes and remove newlines/CR
light_sanitize() {
    local str=$1
    # Remove newlines and carriage returns
    str="${str//$'\n'/}"
    str="${str//$'\r'/}"
    # Escape double quotes
    str="${str//\"/\\\"}"
    echo "$str"
}

# Just to ensure we don't easily break anything, I sadly forbit some special
# characters in git name and e-mails.
# Sorry if it actually breaks someone's way
validate_git_name() {
    local name=$1
    # Allow letters, numbers, spaces, and common punctuation, but no newlines or quotes
    if [[ "$name" =~ [\"$'\n'$'\r'] ]]; then
        error "Invalid git name. Cannot contain quotes or newlines."
    fi
    # Reasonable length limit
    if [[ ${#name} -gt 100 ]]; then
        error "Git name too long (max 100 characters)"
    fi
}

validate_git_email() {
    local email=$1
    # Basic email validation
    if [[ ! "$email" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
        error "Invalid git email format '$email'"
    fi
}

validate_username() {
    local username=$1
    # Unix username constraints
    if [[ ! "$username" =~ ^[a-z_][a-z0-9_-]*$ ]]; then
        error "Invalid username '$username'. Must start with lowercase letter or underscore, followed by lowercase letters, digits, underscores, or hyphens."
    fi
    if [[ ${#username} -gt 32 ]]; then
        error "Username too long (max 32 characters)"
    fi
}

validate_port() {
    local port=$1
    if [[ ! "$port" =~ ^[0-9]+$ ]] || [[ "$port" -lt 1 ]] || [[ "$port" -gt 65535 ]]; then
        error "Invalid port '$port'. Must be a number between 1 and 65535."
    fi
}

validate_uid_gid() {
    local id=$1
    local type=$2
    if [[ ! "$id" =~ ^[0-9]+$ ]] || [[ "$id" -lt 0 ]] || [[ "$id" -gt 65535 ]]; then
        error "Invalid $type '$id'. Must be a number between 0 and 65535."
    fi
}

# Check if base compose exists
check_base_compose() {
    if [[ ! -f "$BASE_COMPOSE" ]]; then
        error "Base compose.yaml not found at $BASE_COMPOSE"
    fi
}

# Get project directory path
get_project_dir() {
    local name=$1
    echo "$PROJECTS_DIR/$name"
}

# Get project config path
get_project_compose() {
    local name=$1
    echo "$PROJECTS_DIR/$name/compose.yaml"
}

# Get project env path
get_project_env() {
    local name=$1
    echo "$PROJECTS_DIR/$name/.env"
}

# I wanted to do some kind of Map to store the plethora of options of a `create`
# command. Turns out it exists since Bash 4+ (2009 AD) as "associative arrays",
# nice!
#
# But wait, MacOS is not relying on Bash 4+, because that ""new version"" (v4.0
# was feb. 2009, Michael Jackson was still alive and Avatar wasn't out yet) had
# a licence change, GPLv3, that has even more copyleft lingo than the v2 of the
# previous versions - maybe that's why they're using zsh now huh?
#
# I could write that script in zsh instead, or in an actual language (probably
# the best idea of all) but I began in bash, bash is (almost) everywhere without
# needing to compile or install some interpreter so it's kind of comfy.
#
# So as often I chose a dumber idea: just implement some kind of dictionary in
# the middle of my simple script which could be 5% of the current LOC amount if
# it was in a modern-er and better-er language.

# Array of keys
declare -a CONFIG_KEYS

# Array of values - Do you see where this is going? ;)
declare -a CONFIG_VALUES

# Add an entry in the config
# Usage: config_set key value
config_set() {
  local key=$1
  local value=$2
  local i=0

  # Check if key exists. If it does, update
  for ((i=0; i<${#CONFIG_KEYS[@]}; i++)); do
    if [[ "${CONFIG_KEYS[$i]}" == "$key" ]]; then
      CONFIG_VALUES[i]="$value"
      return
    fi
  done

  CONFIG_KEYS+=("$key")
  CONFIG_VALUES+=("$value")
}

# Get an entry from the config, or empty string if not found
# Usage: config_get key
config_get() {
    local key=$1
    local i=0

    for ((i=0; i<${#CONFIG_KEYS[@]}; i++)); do
        if [[ "${CONFIG_KEYS[$i]}" == "$key" ]]; then
            echo "${CONFIG_VALUES[$i]}"
            return
        fi
    done

    echo ""
}

# Initialize/reset content of the config to its default value.
config_init() {
    CONFIG_KEYS=()
    CONFIG_VALUES=()

    config_set "host_uid" "$(id -u)"
    config_set "host_gid" "$(id -g)"
    config_set "username" "dev"
    config_set "shell" ""
    config_set "install_node" ""
    config_set "install_rust" ""
    config_set "install_python" ""
    config_set "install_go" ""
    config_set "enable_sudo" ""
    config_set "git_name" ""
    config_set "git_email" ""
    config_set "packages" ""
    config_set "install_neovim" ""
    config_set "install_starship" ""
    config_set "install_atuin" ""
    config_set "install_mise" ""
    config_set "install_zellij" ""
    config_set "project_path" ""
    config_set "prompted" "false"
}

# Generate project compose file
# Usage: generate_project_compose name ports_array volumes_array
generate_project_compose() {
    local name=$1
    shift

    # Remaining is ports + "VOLUMES_START" + volumes
    local ports=("$@")

    # Find where volumes start
    local volumes=()
    local in_volumes=0
    local temp_ports=()

    for item in "${ports[@]}"; do
        if [[ "$item" == "VOLUMES_START" ]]; then
            in_volumes=1
            continue
        fi

        if [[ $in_volumes -eq 0 ]]; then
            temp_ports+=("$item")
        else
            volumes+=("$item")
        fi
    done

    ports=("${temp_ports[@]}")

    local compose_file
    local env_file
    compose_file=$(get_project_compose "$name")
    env_file=$(get_project_env "$name")

    if [[ -f "$compose_file" || -f "$env_file" ]]; then
        error "Project '$name' already exists. You can have multiple configurations for the same project by calling 'create' with the '--name' flag. Hint: Use 'paul-envs.sh list' to see all projects or 'paul-envs.sh remove $name' to delete it"
    fi

    # Sanitize all user inputs
    local safe_git_name
    safe_git_name=$(light_sanitize "$(config_get git_name)")
    local safe_git_email
    safe_git_email=$(light_sanitize "$(config_get git_email)")
    local safe_packages
    safe_packages=$(light_sanitize "$(config_get packages)")

    mkdir -p "$(dirname "$compose_file")"

    # Generate .env file
    cat >> "$env_file" <<EOF
# "Env file" for your project, which will be fed to \`docker compose\`
# alongside "compose.yaml" in the same directory.
#
# Can be freely updated, with the condition of not removing a few
# mandatory env values:
# - PROJECT_NAME
# - PROJECT_PATH

# Name of the project directory inside the container.
PROJECT_NAME="${name}"

# Path to the project you want to mount in this container
# Will be mounted in "\$HOME/projects/<PROJECT_NAME>" inside that container.
PROJECT_PATH="$(config_get project_path)"

# To align with your current uid.
# This is to ensure the mounted volume from your host has compatible
# permissions.
# On POSIX-like systems, just run \`id -u\` with the wanted user to know it.
HOST_UID="$(config_get host_uid)"

# To align with your current gid (same reason than for "uid").
# On POSIX-like systems, just run \`id -g\` with the wanted user to know it.
HOST_GID="$(config_get host_gid)"

# Username created in the container.
# Not really important, just set it if you want something other than "dev".
USERNAME="$(config_get username)"

# The default shell wanted.
# Only "bash", "zsh" or "fish" are supported for now.
USER_SHELL="$(config_get shell)"

# Whether to install Node.js, and the version wanted.
# Note that a WebAssembly target is also automatically ready.
#
# Values can be:
# - if \`none\`: don't install Node.js
# - if \`latest\`: Install Ubuntu's default package for Node.js
# - If anything else: The exact version to install (e.g. "1.90.0").
#   That last type of value will only work if \`INSTALL_MISE\` is \`true\`.
INSTALL_NODE="$(config_get install_node)"

# Whether to install Rust and Cargo, and the version wanted.
# Note that a WebAssembly target is also automatically ready.
#
# Values can be:
# - if \`none\`: don't install Rust
# - if \`latest\`: Install Ubuntu's default package for Rust
#   Ubuntu base's repositories
# - If anything else: The exact version to install (e.g. "1.90.0").
#   That last type of value will only work if \`INSTALL_MISE\` is \`true\`.
INSTALL_RUST="$(config_get install_rust)"

# Whether to install Python, and the version wanted.
#
# Values can be:
# - if \`none\`: don't install Python
# - if \`latest\`: Install Ubuntu's default package for Python
# - If anything else: The exact version to install (e.g. "3.12.0").
#   That last type of value will only work if \`INSTALL_MISE\` is \`true\`.
INSTALL_PYTHON="$(config_get install_python)"

# Whether to install Go, and the version wanted.
# Note that GOPATH is automatically set to ~/go
#
# Values can be:
# - if \`none\`: don't install Go
# - if \`latest\`: Install Ubuntu's default package for Go
# - If anything else: The exact version to install (e.g. "1.21.5").
#   That last type of value will only work if \`INSTALL_MISE\` is \`true\`.
INSTALL_GO="$(config_get install_go)"

# If \`true\`, \`sudo\` will be installed, passwordless.
ENABLE_SUDO="$(config_get enable_sudo)"

# Additional packages outside the core base, separated by a space.
# Have to be in Ubuntu's default repository
# (e.g. "ripgrep fzf". Can be left empty for no supplementary packages)
SUPPLEMENTARY_PACKAGES="$safe_packages"

# Tools toggle.
# "true" == install it
# anything else == don't.
INSTALL_NEOVIM="$(config_get install_neovim)"
INSTALL_STARSHIP="$(config_get install_starship)"
INSTALL_ATUIN="$(config_get install_atuin)"
INSTALL_MISE="$(config_get install_mise)"
INSTALL_ZELLIJ="$(config_get install_zellij)"

# Git author and committer name used inside the container
# Can also be empty to not set that in the container.
GIT_AUTHOR_NAME="$safe_git_name"

# Git author and committer e-mail used inside the container
# Can also be empty to not set that in the container.
GIT_AUTHOR_EMAIL="$safe_git_email"
EOF

    # Generate YAML
    cat >> "$compose_file" <<EOF
# Compose file for your project, which will be fed to \`docker compose\` alongside
# ".env" in the same directory.
#
# Can be freely updated to update ports, volumes etc.
services:
  paulenv:
    build:
EOF

    # Add ports if specified
    if [[ ${#ports[@]} -gt 0 ]]; then
        echo "    ports:" >> "$compose_file"
        for port in "${ports[@]}"; do
            echo "      - \"$port:$port\"" >> "$compose_file"
        done
    fi

    # Add volumes if wanted
    cat >> "$compose_file" <<EOF
    volumes:
EOF
    # TODO: some kind of pre-validation?
    for vol in "${volumes[@]}"; do
        echo "      - $vol" >> "$compose_file"
    done

    echo "" >> "$compose_file"
}

# Prompt for shell if not set
prompt_shell() {
    if [[ -n "$(config_get shell)" ]]; then
        return
    fi

    echo ""
    info "=== Shell Selection ==="
    echo "Select shell:"
    echo "  1) bash (default)"
    echo "  2) zsh"
    echo "  3) fish"
    read -r -p "Choice [1]: " shell_choice
    case ${shell_choice:-1} in
        1) config_set "shell" "bash" ;;
        2) config_set "shell" "zsh" ;;
        3) config_set "shell" "fish" ;;
        *) config_set "shell" "bash" ;;
    esac
}

# Prompt for language runtimes if not set
prompt_languages() {
    local has_any_lang=0

    # Check if any language was explicitly set
    if [[ -n "$(config_get install_node)" ]] || \
       [[ -n "$(config_get install_rust)" ]] || \
       [[ -n "$(config_get install_python)" ]] || \
       [[ -n "$(config_get install_go)" ]]; then
        has_any_lang=1
    fi

    if [[ $has_any_lang -eq 1 ]]; then
        # Set defaults for unspecified languages
        [[ -z "$(config_get install_node)" ]] && config_set "install_node" "none"
        [[ -z "$(config_get install_rust)" ]] && config_set "install_rust" "none"
        [[ -z "$(config_get install_python)" ]] && config_set "install_python" "none"
        [[ -z "$(config_get install_go)" ]] && config_set "install_go" "none"
        return
    fi

    echo ""
    info "=== Language Runtimes ==="
    echo "Which language runtimes do you need? (space-separated numbers, or Enter to skip)"
    echo "  1) Node.js"
    echo "  2) Rust"
    echo "  3) Python"
    echo "  4) Go"
    read -r -p "Choice [none]: " lang_choices

    # Set all to none first
    config_set "install_node" "none"
    config_set "install_rust" "none"
    config_set "install_python" "none"
    config_set "install_go" "none"

    for choice in $lang_choices; do
        case $choice in
            1)
                read -r -p "Node.js version (latest/none/X.Y.Z) [latest]: " node_ver
                config_set "install_node" "${node_ver:-latest}"
                ;;
            2)
                read -r -p "Rust version (latest/none/X.Y.Z) [latest]: " rust_ver
                config_set "install_rust" "${rust_ver:-latest}"
                ;;
            3)
                read -r -p "Python version (latest/none/X.Y.Z) [latest]: " python_ver
                config_set "install_python" "${python_ver:-latest}"
                ;;
            4)
                read -r -p "Go version (latest/none/X.Y.Z) [latest]: " go_ver
                config_set "install_go" "${go_ver:-latest}"
                ;;
            *)
                warn "Unknown choice: $choice (skipped)"
                ;;
        esac
    done
}

# Prompt for supplementary packages if not set
prompt_packages() {
    if [[ -n "$(config_get prompted)" && "$(config_get prompted)" == "true" ]]; then
        # Already prompted via flag
        return
    fi

    echo ""
    info "=== Additional Packages ==="
    echo "Enter additional Ubuntu packages (space-separated, or Enter to skip):"
    echo "Examples: ripgrep fzf bat htop"
    read -r -p "Packages: " packages

    if [[ -n "$packages" ]]; then
        validate_apt_package_names "$packages"
        config_set "packages" "$packages"
    fi
}

# Prompt for tools if not set
prompt_tools() {
    local tools_set=0

    # Check if any tool was explicitly set
    if [[ -n "$(config_get install_neovim)" ]] || \
       [[ -n "$(config_get install_starship)" ]] || \
       [[ -n "$(config_get install_atuin)" ]] || \
       [[ -n "$(config_get install_mise)" ]] || \
       [[ -n "$(config_get install_zellij)" ]]; then
        tools_set=1
    fi

    if [[ $tools_set -eq 1 ]]; then
        # Set defaults for unspecified tools
        [[ -z "$(config_get install_neovim)" ]] && config_set "install_neovim" "true"
        [[ -z "$(config_get install_starship)" ]] && config_set "install_starship" "true"
        [[ -z "$(config_get install_atuin)" ]] && config_set "install_atuin" "true"
        [[ -z "$(config_get install_mise)" ]] && config_set "install_mise" "true"
        [[ -z "$(config_get install_zellij)" ]] && config_set "install_zellij" "true"
        return
    fi

    echo ""
    info "=== Development Tools ==="
    echo "Which tools do you want to install? (space-separated numbers, or Enter for all)"
    echo "  1) Neovim (text editor)"
    echo "  2) Starship (prompt)"
    echo "  3) Atuin (shell history)"
    echo "  4) Mise (version manager)"
    echo "  5) Zellij (terminal multiplexer)"
    read -r -p "Choice [all]: " tool_choices

    # Default to all if empty
    if [[ -z "$tool_choices" ]]; then
        tool_choices="1 2 3 4 5"
    fi

    # Set all to false first
    config_set "install_neovim" "false"
    config_set "install_starship" "false"
    config_set "install_atuin" "false"
    config_set "install_mise" "false"
    config_set "install_zellij" "false"

    for choice in $tool_choices; do
        case $choice in
            1) config_set "install_neovim" "true" ;;
            2) config_set "install_starship" "true" ;;
            3) config_set "install_atuin" "true" ;;
            4) config_set "install_mise" "true" ;;
            5) config_set "install_zellij" "true" ;;
            *)
                warn "Unknown choice: $choice (skipped)"
                ;;
        esac
    done
}

# Prompt for sudo if not set
prompt_sudo() {
    if [[ -n "$(config_get enable_sudo)" ]]; then
        return
    fi

    echo ""
    info "=== Sudo Access ==="
    read -r -p "Enable sudo access in container? (y/N): " sudo_choice
    if [[ $sudo_choice =~ ^[Yy]$ ]]; then
        config_set "enable_sudo" "true"
    else
        config_set "enable_sudo" "false"
    fi
}

# Prompt for ports if not set
prompt_ports() {
    local ports_var="$1"

    echo ""
    info "=== Port Forwarding ==="
    echo "Enter ports to expose (space-separated, or Enter to skip):"
    echo "Examples: 3000 5432 8080"
    read -r -p "Ports: " port_input

    for port in $port_input; do
        validate_port "$port"
        eval "$ports_var+=('$port')"
    done
}

# Prompt for volumes/credentials
prompt_volumes() {
    local volumes_var="$1"

    echo ""
    info "=== Credentials & Volumes ==="
    echo "Mount common credentials/configs? (space-separated numbers, or Enter to skip)"
    echo "  1) SSH keys (~/.ssh)"
    echo "  2) Git credentials (~/.git-credentials)"
    echo "  3) AWS credentials (~/.aws)"
    echo "  4) Custom CA certificates (/etc/ssl/certs/custom-ca.crt)"
    read -r -p "Choice [none]: " choices

    for choice in $choices; do
        case $choice in
            1)
                eval "$volumes_var+=('$HOME/.ssh:/home/\${USERNAME}/.ssh:ro')"
                ;;
            2)
                eval "$volumes_var+=('$HOME/.git-credentials:/home/\${USERNAME}/.git-credentials:ro')"
                ;;
            3)
                eval "$volumes_var+=('$HOME/.aws:/home/\${USERNAME}/.aws:ro')"
                ;;
            4)
                eval "$volumes_var+=('/etc/ssl/certs/custom-ca.crt:/usr/local/share/ca-certificates/custom-ca.crt:ro')"
                ;;
            *)
                warn "Unknown choice: $choice (skipped)"
                ;;
        esac
    done

    echo ""
    echo "Add custom volumes? (one per line, Enter on empty line to finish)"
    echo "Format: /host/path:/container/path[:ro]"
    while true; do
        read -r -p "Volume: " vol
        if [[ -z "$vol" ]]; then
            break
        fi
        eval "$volumes_var+=('$vol')"
    done
}

# Commands
cmd_create() {
    config_init

    local name=""
    local project_path=""
    local ports=()
    local volumes=()
    local no_prompt=0

    # First two positional args
    if [[ $# -lt 1 ]]; then
        error "Usage: paul-envs.sh create <project-path> [options]"
    fi

    project_path=$1
    config_set "project_path" "$project_path"
    shift 1

    # Parse flags
    while [[ $# -gt 0 ]]; do
        case $1 in
            --no-prompt)
                no_prompt=1
                shift
                ;;
            --name)
                # `name` is validated below
                name="$2"
                shift 2
                ;;
            --uid)
                validate_uid_gid "$2" "UID"
                config_set "host_uid" "$2"
                shift 2
                ;;
            --gid)
                validate_uid_gid "$2" "GID"
                config_set "host_gid" "$2"
                shift 2
                ;;
            --username)
                validate_username "$2"
                config_set "username" "$2"
                shift 2
                ;;
            --shell)
                validate_shell "$2"
                config_set "shell" "$2"
                shift 2
                ;;
            --nodejs)
                validate_version_arg "$2"
                config_set "install_node" "$2"
                shift 2
                ;;
            --rust)
                validate_version_arg "$2"
                config_set "install_rust" "$2"
                shift 2
                ;;
            --python)
                validate_version_arg "$2"
                config_set "install_python" "$2"
                shift 2
                ;;
            --go)
                validate_version_arg "$2"
                config_set "install_go" "$2"
                shift 2
                ;;
            --enable-sudo|--sudo)
              config_set "enable_sudo" "true"
              shift
              ;;
            --git-name)
                validate_git_name "$2"
                config_set "git_name" "$2"
                shift 2
                ;;
            --git-email)
                validate_git_email "$2"
                config_set "git_email" "$2"
                shift 2
                ;;
            --packages)
                validate_apt_package_names "$2"
                config_set "packages" "$2"
                shift 2
                ;;
            --no-neovim)
                config_set "install_neovim" "false"
                shift
                ;;
            --no-starship)
                config_set "install_starship" "false"
                shift
                ;;
            --no-atuin)
                config_set "install_atuin" "false"
                shift
                ;;
            --no-mise)
                config_set "install_mise" "false"
                shift
                ;;
            --no-zellij)
                config_set "install_zellij" "false"
                shift
                ;;
            --port)
                validate_port "$2"
                ports+=("$2")
                shift 2
                ;;
            --volume)
                # Volumes are passed to docker-compose which validates them
                # We just store them as-is
                volumes+=("$2")
                shift 2
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done

    # If --no-prompt, validate we have everything needed
    if [[ $no_prompt -eq 1 ]]; then
        # Set defaults for anything not specified
        [[ -z "$(config_get shell)" ]] && config_set "shell" "bash"
        [[ -z "$(config_get install_node)" ]] && config_set "install_node" "none"
        [[ -z "$(config_get install_rust)" ]] && config_set "install_rust" "none"
        [[ -z "$(config_get install_python)" ]] && config_set "install_python" "none"
        [[ -z "$(config_get install_go)" ]] && config_set "install_go" "none"
        [[ -z "$(config_get enable_sudo)" ]] && config_set "enable_sudo" "false"
        [[ -z "$(config_get install_neovim)" ]] && config_set "install_neovim" "true"
        [[ -z "$(config_get install_starship)" ]] && config_set "install_starship" "true"
        [[ -z "$(config_get install_atuin)" ]] && config_set "install_atuin" "true"
        [[ -z "$(config_get install_mise)" ]] && config_set "install_mise" "true"
        [[ -z "$(config_get install_zellij)" ]] && config_set "install_zellij" "true"
    else
        # Interactive mode - prompt for missing values
        prompt_shell
        prompt_languages
        prompt_packages
        prompt_tools
        prompt_sudo

        # Only prompt for ports if none were specified
        if [[ ${#ports[@]} -eq 0 ]]; then
            prompt_ports ports
        fi

        # Only prompt for volumes if none were specified
        if [[ ${#volumes[@]} -eq 0 ]]; then
            prompt_volumes volumes
        fi
    fi

    # Determine project name
    if [[ -z "$name" ]]; then
        name="$(basename "$(config_get project_path)")"
    fi
    validate_project_name "$name"

    # Validate path exists or warn
    mkdir -p "$PROJECTS_DIR"

    local final_path
    final_path="$(config_get project_path)"

    if [[ "$(config_get install_mise)" != "true" ]]; then
      warn "\`mise\` is not installed. We will use Ubuntu's repositories for language runtime versions, if needed."
    fi

    if [[ ! -d "$final_path" && $no_prompt -eq 0 ]]; then
        warn "Warning: Path $final_path does not exist"
        read -p "Create config anyway? (y/N) " -n 1 -r
        echo
        [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
    fi

    # Generate config
    generate_project_compose "$name" "${ports[@]}" "VOLUMES_START" "${volumes[@]}"

    success "Created project '$name'"
    echo ""
    echo "Next steps:"
    echo "  1. Review/edit configuration:"
    echo "     - $(get_project_env "$name")"
    echo "     - $(get_project_compose "$name")"
    echo "  2. Build the environment:"
    echo "     paul-envs.sh build $name"
    echo "  3. Run the environment:"
    echo "     paul-envs.sh run $name"
}

cmd_list() {
    check_base_compose
    if [[ ! -d "$PROJECTS_DIR" ]]; then
        echo "No project created yet"
        echo "Hint: Create one with 'paul-envs.sh create <path>'"
        exit 0
    fi

    echo "Projects created:"
    local found=0
    for dir in "$PROJECTS_DIR"/*; do
        if [[ -d "$dir" && -f "$dir/compose.yaml" ]]; then
            found=1
            name=$(basename "$dir")
            path=$(grep "PROJECT_PATH=" "$dir"/.env | head -1 | sed 's/PROJECT_PATH=//' | tr -d '"')
            echo "  - $name"
            echo "      Path: $path"
        fi
    done

    if [[ $found -eq 0 ]]; then
        echo "  (no project found)"
        echo "Hint: Create one with 'paul-envs.sh create <path>'"
    fi
}

cmd_build() {
    check_base_compose
    local name=$1

    if [[ -z "$name" ]]; then
        error "Usage: paul-envs.sh build <name>\nHint: Use 'paul-envs.sh list' to see available projects"
    fi

    validate_project_name "$name"

    local compose_file
    local env_file
    compose_file=$(get_project_compose "$name")
    env_file=$(get_project_env "$name")
    if [[ ! -f "$compose_file" || ! -f "$env_file" ]]; then
        error "Project '$name' not found. Hint: Use 'paul-envs.sh list' to see available projects or 'paul-envs.sh create' to make a new one"
    fi

    # Ensure shared cache volume exists
    docker volume create paulenv-shared-cache 2>/dev/null || true

    export COMPOSE_PROJECT_NAME="paulenv-$name"
    docker compose -f "$BASE_COMPOSE" -f "$compose_file" --env-file "$env_file" build
    success "Built project '$name'"

    echo ""
    warn "Resetting persistent volumes..."
    docker compose -f "$BASE_COMPOSE" -f "$compose_file" --env-file "$env_file" --profile reset up reset-cache reset-local
    success "Volumes reset complete"
}

cmd_run() {
    check_base_compose
    local name=$1
    shift 1

    if [[ -z "$name" ]]; then
        error "Usage: paul-envs.sh run <name>\nHint: Use 'paul-envs.sh list' to see available projects"
    fi

    validate_project_name "$name"

    local compose_file
    local env_file
    compose_file=$(get_project_compose "$name")
    env_file=$(get_project_env "$name")
    if [[ ! -f "$compose_file" || ! -f "$env_file" ]]; then
        error "Project '$name' not found\nHint: Use 'paul-envs.sh list' to see available projects"
    fi

    export COMPOSE_PROJECT_NAME="paulenv-$name"
    docker compose -f "$BASE_COMPOSE" -f "$compose_file" --env-file "$env_file" run --rm paulenv
}

cmd_remove() {
    local name=$1

    if [[ -z "$name" ]]; then
        error "Usage: paul-envs.sh remove <name>\nHint: Use 'paul-envs.sh list' to see available projects"
    fi

    validate_project_name "$name"

    local project_dir
    project_dir=$(get_project_dir "$name")
    if [[ ! -d "$project_dir" ]]; then
        error "Project '$name' not found\nHint: Use 'paul-envs.sh list' to see available projects"
    fi

    read -p "Remove project '$name'? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$project_dir"
        success "Removed project '$name'"
        echo "Note: Docker volumes are preserved. To remove them, run:"
        echo "  docker volume rm paulenv-$name-local"
    fi
}

# Main command dispatcher
case ${1:-} in
    create)
        shift
        cmd_create "$@"
        ;;
    list|ls)
        shift
        cmd_list "$@"
        ;;
    build)
        shift
        cmd_build "$@"
        ;;
    run)
        shift
        cmd_run "$@"
        ;;
    remove|rm)
        shift
        cmd_remove "$@"
        ;;
    *)
        cat <<EOF
paul-envs.sh - Development Environment Manager

Usage:
  paul-envs.sh create <path> [options]
  paul-envs.sh list
  paul-envs.sh build <name>
  paul-envs.sh run <name> [command]
  paul-envs.sh remove <name>

Options for create:
  --no-prompt              Non-interactive mode (uses defaults)
  --name NAME              Name of this project (default: directory name)
  --uid UID                Host UID (default: current user)
  --gid GID                Host GID (default: current group)
  --username NAME          Container username (default: dev)
  --shell SHELL            User shell: bash|zsh|fish (prompted if not specified)
  --nodejs VERSION         Node.js installation:
                             'none' - skip installation of Node.js
                             'latest' - use Ubuntu default package
                             '20.10.0' - specific version (via mise)
                           (prompted if not specified)
  --rust VERSION           Rust installation:
                             'none' - skip installation of Rust
                             'latest' - latest stable via rustup
                             '1.75.0' - specific version (via mise)
                           (prompted if not specified)
  --python VERSION         Python installation:
                             'none' - skip installation of Python
                             'latest' - use Ubuntu default package
                             '3.12.0' - specific version (via mise)
                           (prompted if not specified)
  --go VERSION             Go installation:
                             'none' - skip installation of Go
                             'latest' - use Ubuntu default package
                             '1.21.5' - specific version (via mise)
                           (prompted if not specified)
  --enable-sudo            Enable sudo access in container (prompted if not specified)
  --git-name NAME          Git user.name (optional)
  --git-email EMAIL        Git user.email (optional)
  --packages "PKG1 PKG2"   Additional Ubuntu packages (prompted if not specified)
  --no-neovim              Skip Neovim installation
  --no-starship            Skip Starship prompt installation
  --no-atuin               Skip Atuin shell history installation
  --no-mise                Skip Mise tool manager installation
  --no-zellij              Skip Zellij terminal multiplexer installation
  --port PORT              Expose container port (prompted if not specified, can be repeated)
  --volume HOST:CONT[:ro]  Mount volume (prompted if not specified, can be repeated)

Interactive Mode (default):
  paul-envs.sh create ~/projects/myapp
  # Will prompt for all unspecified options

Non-Interactive Mode:
  paul-envs.sh create ~/projects/myapp --no-prompt --shell bash --nodejs latest

Mixed Mode (some flags + prompts):
  paul-envs.sh create ~/projects/myapp --nodejs 20.10.0 --rust latest
  # Will prompt for shell, packages, tools, sudo, ports, and volumes

Full Configuration Example:
  paul-envs.sh create ~/work/api \\
    --name myApp \\
    --shell zsh \\
    --nodejs 20.10.0 \\
    --rust latest \\
    --python 3.12.0 \\
    --go latest \\
    --enable-sudo \\
    --git-name "John Doe" \\
    --git-email "john@example.com" \\
    --packages "ripgrep fzf bat" \\
    --no-atuin \\
    --port 3000 \\
    --port 5432 \\
    --volume ~/.git-credentials:/home/dev/.git-credentials:ro

Configuration:
  Base compose: $BASE_COMPOSE
  Projects directory: $PROJECTS_DIR
EOF
        ;;
esac
