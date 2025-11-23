#!/bin/bash
set -e

# Detect if running on Windows (Git Bash, MSYS2, Cygwin, WSL)
detect_windows() {
    if [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]] || [[ "$OSTYPE" == "win32" ]]; then
        return 0
    elif [[ -n "$WINDIR" ]] || [[ -n "$SYSTEMROOT" ]]; then
        return 0
    fi
    return 1
}

IS_WINDOWS=0
if detect_windows; then
    IS_WINDOWS=1
fi

# Convert Windows paths to Unix paths for Docker
normalize_path() {
    local path="$1"

    if [[ $IS_WINDOWS -eq 1 ]]; then
        # Handle Windows drive letters (C:\ -> /c/)
        if [[ "$path" =~ ^([A-Za-z]):[\\/](.*)$ ]]; then
            local drive="${BASH_REMATCH[1]}"
            local rest="${BASH_REMATCH[2]}"
            drive=$(echo "$drive" | tr '[:upper:]' '[:lower:]')
            rest="${rest//\\//}"
            echo "/$drive/$rest"
        # Handle Git Bash paths (/c/Users/...)
        elif [[ "$path" =~ ^/([a-z])/(.*)$ ]]; then
            echo "$path"
        # Handle relative paths
        else
            path="${path//\\//}"
            echo "$path"
        fi
    else
        echo "$path"
    fi
}

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
    if [[ "$SCRIPT_PATH" != /* ]]; then
        SCRIPT_PATH="$SCRIPT_DIR/$SCRIPT_PATH"
    fi
done
SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
SCRIPT_DIR=$(normalize_path "$SCRIPT_DIR")
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

get_absolute_path() {
    local path="$1"

    # Normalize backslashes first
    path="${path//\\//}"

    # Try realpath -m first (allows non-existent paths)
    if command -v realpath &> /dev/null; then
        local result
        result=$(realpath -m "$path" 2>/dev/null)
        if [[ -n "$result" ]]; then
            normalize_path "$result"
            return 0
        fi
    fi

    # Fallback: manual resolution
    # Handle absolute paths (Windows or Unix)
    if [[ "$path" =~ ^[A-Za-z]:[\\/] ]]; then
        # Windows absolute path
        normalize_path "$path"
        return 0
    elif [[ "$path" = /* ]]; then
        # Unix absolute path
        normalize_path "$path"
        return 0
    fi

    # Handle relative paths - prepend current directory
    local cwd
    cwd="$(pwd)"
    cwd=$(normalize_path "$cwd")
    echo "$cwd/$path"
}

# Security validation functions
validate_project_name() {
    local name=$1
    if [[ -z "$name" ]] || [[ ! "$name" =~ ^[a-zA-Z0-9_][a-zA-Z0-9_-]{0,127}$ ]]; then
        error "Invalid project name '$name'. Must be 1-128 characters, start with alphanumeric or underscore, and contain only alphanumeric, hyphens, and underscores."
    fi
}

# Sanitize project name for Docker image tag
sanitize_project_name() {
    local input="$1"
    local sanitized

    # Convert to lowercase (portable)
    sanitized="$(echo "$input" | tr '[:upper:]' '[:lower:]')"

    # Replace invalid characters with hyphens
    sanitized="${sanitized//[^a-z0-9_-]/-}"

    # Remove leading non-alphanumeric characters
    while [[ "$sanitized" =~ ^[^a-z0-9] ]]; do
        sanitized="${sanitized:1}"
    done

    # Collapse multiple consecutive hyphens
    while [[ "$sanitized" == *"--"* ]]; do
        sanitized="${sanitized//--/-}"
    done

    # Truncate to 128 characters
    sanitized="${sanitized:0:128}"

    # Remove trailing hyphens
    while [[ "$sanitized" == *"-" ]]; do
        sanitized="${sanitized%-}"
    done

    # Ensure not empty
    if [[ -z "$sanitized" ]]; then
        sanitized="project"
    fi

    echo "$sanitized"
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

    # On Windows, use fixed UID/GID suitable for Docker
    if [[ $IS_WINDOWS -eq 1 ]]; then
        config_set "host_uid" "1000"
        config_set "host_gid" "1000"
    else
        config_set "host_uid" "$(id -u)"
        config_set "host_gid" "$(id -g)"
    fi

    config_set "username" "dev"
    config_set "shell" ""
    config_set "install_node" ""
    config_set "install_rust" ""
    config_set "install_python" ""
    config_set "install_go" ""
    config_set "enable_wasm" ""
    config_set "enable_ssh" ""
    config_set "enable_sudo" ""
    config_set "git_name" ""
    config_set "git_email" ""
    config_set "packages" ""
    config_set "install_neovim" ""
    config_set "install_starship" ""
    config_set "install_atuin" ""
    config_set "install_mise" ""
    config_set "install_zellij" ""
    config_set "install_jujutsu" ""
    config_set "project_host_path" ""
}

# Check that the given name does not have already project files. Exits on error if so.
# Usage: check_inexistent_name name
check_inexistent_name() {
    local compose_file
    local env_file
    compose_file=$(get_project_compose "$1")
    env_file=$(get_project_env "$1")

    if [[ -f "$compose_file" || -f "$env_file" ]]; then
        error "Project '$name' already exists. You can have multiple configurations for the same project by calling 'create' with the '--name' flag. Hint: Use 'paul-envs.sh list' to see all projects or 'paul-envs.sh remove $name' to delete it"
    fi
}

# Check if specific versions were requested but mise is not installed
# usage mise_check no_prompt
mise_check() {
    local needs_mise_warning=0
    if does_lang_version_needs_mise "$(config_get install_node)" || \
       does_lang_version_needs_mise "$(config_get install_rust)" || \
       does_lang_version_needs_mise "$(config_get install_python)" || \
       does_lang_version_needs_mise "$(config_get install_go)"; then
        if [[ "$(config_get install_mise)" != "true" ]]; then
            needs_mise_warning=1
        fi
    fi

    if [[ $needs_mise_warning -eq 1 ]]; then
        echo ""
        warn "WARNING: You specified exact version(s) for language runtimes, but Mise is not enabled."
        warn "Exact versions require Mise to be installed. Without Mise, Ubuntu's default packages will be used instead."
        if [[ $1 -eq 0 ]]; then
            read -r -p "Would you like to enable Mise now? (Y/n): " mise_choice
            if [[ ! $mise_choice =~ ^[Nn]$ ]]; then
                config_set "install_mise" "true"
                success "Mise enabled"
            fi
        fi
    fi
}

# Check if a specific version requires mise
does_lang_version_needs_mise() {
    local version=$1
    if [[ "$version" != "none" && "$version" != "latest" && "$version" != "" ]]; then
        return 0
    fi
    return 1
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

    check_inexistent_name "$name"

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
# Can be freely updated.

# Uniquely identify this container.
# *SHOULD NOT BE UPDATED*
PROJECT_ID="$name"

# Name of the project directory inside the container.
# A \`PROJECT_DIRNAME\` should always be set
PROJECT_DIRNAME="$(config_get project_dest_path)"

# Path to the project you want to mount in this container
# Will be mounted in "\$HOME/projects/<PROJECT_DIRNAME>" inside that container.
# A \`PROJECT_PATH\` should always be set
PROJECT_PATH="$(config_get project_host_path)"

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

# If \`true\`, add WebAssembly-specialized tools such as \`binaryen\` and a
# WebAssembly target for Rust if it is installed.
ENABLE_WASM="$(config_get enable_wasm)"

# If \`true\`, \`openssh\` will be installed, and the container will listen for ssh
# connections at port 22.
ENABLE_SSH="$(config_get enable_ssh)"

# If \`true\`, \`sudo\` will be installed, with a password set to "dev".
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
INSTALL_JUJUTSU="$(config_get install_jujutsu)"

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
    image: paulenv:$name
EOF

    # Add ports if specified
    if [[ ${#ports[@]} -gt 0 ]]; then
        echo "    ports:" >> "$compose_file"
        for port in "${ports[@]}"; do
            echo "      - \"$port:$port\"" >> "$compose_file"
        done
        if [[ "$(config_get enable_ssh)" = "true" ]]; then
          echo "      # to listen for ssh connections" >> "$compose_file"
          echo "      - \"22:22\"" >> "$compose_file"
        fi
    elif [[ "$(config_get enable_ssh)" = "true" ]]; then
        cat >> "$compose_file" <<EOF
    ports:
      # to listen for ssh connections
      - "22:22"
EOF
    fi

    # Add volumes if wanted
    cat >> "$compose_file" <<EOF
    volumes:
EOF
    # TODO: some kind of pre-validation?
    for vol in "${volumes[@]}"; do
        echo "      - $vol" >> "$compose_file"
    done

    if [[ "$(config_get enable_ssh)" = "true" ]]; then
        local ssh_key_path
        ssh_key_path="$(config_get ssh_key_path)"

        if [[ -n "$ssh_key_path" && -f "$ssh_key_path" ]]; then
            ssh_key_path=$(normalize_path "$ssh_key_path")
            echo "      # Your local public key for ssh:" >> "$compose_file"
            echo "      - $ssh_key_path:/etc/ssh/authorized_keys/\${USERNAME}:ro" >> "$compose_file"
        else
            warn "No SSH key configured"
            warn "Adding a note to your compose file: $compose_file"
            echo "      # Add your SSH public key here, for example:" >> "$compose_file"
            echo "      # - ~/.ssh/id_ed25519.pub:/etc/ssh/authorized_keys/\${USERNAME}:ro" >> "$compose_file"
        fi
    fi
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
        if [[ -z "$(config_get install_node)" ]]; then
            config_set "install_node" "none"
        fi
        if [[ -z "$(config_get install_rust)" ]]; then
            config_set "install_rust" "none"
        fi
        if [[ -z "$(config_get install_python)" ]]; then
            config_set "install_python" "none"
        fi
        if [[ -z "$(config_get install_go)" ]]; then
            config_set "install_go" "none"
        fi
        return
    fi

    echo ""
    info "=== Language Runtimes ==="
    echo "Which language runtimes do you need? (space-separated numbers, or Enter to skip)"
    echo "  1) Node.js"
    echo "  2) Rust"
    echo "  3) Python"
    echo "  4) Go"
    echo "  5) WebAssembly tools (Binaryen, Rust WASM target if Rust is enabled)"
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
            5)
                config_set "enable_wasm" "true"
                ;;
            *)
                warn "Unknown choice: $choice (skipped)"
                ;;
        esac
    done
}

# Prompt for supplementary packages if not set
prompt_packages() {
    if [[ -n "$(config_get packages)" ]]; then
        # Already explicitly set
        return
    fi

    echo ""
    info "=== Additional Packages ==="
    echo "The following packages are already installed on top of an Ubuntu:24.04 image:"
    echo "curl git build-essential"
    echo ""
    echo "Enter additional Ubuntu packages (space-separated, or Enter to skip):"
    echo "Examples: ripgrep fzf htop"
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
       [[ -n "$(config_get install_zellij)" ]] || \
       [[ -n "$(config_get install_jujutsu)" ]]; then
        tools_set=1
    fi

    if [[ $tools_set -eq 1 ]]; then
        # Set defaults for unspecified tools
        if [[ -z "$(config_get install_neovim)" ]]; then
            config_set "install_neovim" "false"
        fi
        if [[ -z "$(config_get install_starship)" ]]; then
            config_set "install_starship" "false"
        fi
        if [[ -z "$(config_get install_atuin)" ]]; then
            config_set "install_atuin" "false"
        fi
        if [[ -z "$(config_get install_mise)" ]]; then
            config_set "install_mise" "false"
        fi
        if [[ -z "$(config_get install_zellij)" ]]; then
            config_set "install_zellij" "false"
        fi
        if [[ -z "$(config_get install_jujutsu)" ]]; then
            config_set "install_jujutsu" "false"
        fi
        return
    fi

    echo ""
    info "=== Development Tools ==="
    echo "Some dev tools are not pulled from Ubuntu's repositories to get their latest version instead."
    echo "Which of those tools do you want to install? (space-separated numbers, or Enter to skip all)"
    echo "  1) Neovim (text editor)"
    echo "  2) Starship (prompt)"
    echo "  3) Atuin (shell history)"
    echo "  4) Mise (version manager - required for specific language versions)"
    echo "  5) Zellij (terminal multiplexer)"
    echo "  6) Jujutsu (Git-compatible VCS)"
    read -r -p "Choice [none]: " tool_choices

    # Set all to false first
    config_set "install_neovim" "false"
    config_set "install_starship" "false"
    config_set "install_atuin" "false"
    config_set "install_mise" "false"
    config_set "install_zellij" "false"
    config_set "install_jujutsu" "false"

    for choice in $tool_choices; do
        case $choice in
            1) config_set "install_neovim" "true" ;;
            2) config_set "install_starship" "true" ;;
            3) config_set "install_atuin" "true" ;;
            4) config_set "install_mise" "true" ;;
            5) config_set "install_zellij" "true" ;;
            6) config_set "install_jujutsu" "true" ;;
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
    read -r -p "Enable sudo access in container (password:\"dev\")? (y/N): " sudo_choice
    if [[ $sudo_choice =~ ^[Yy]$ ]]; then
        config_set "enable_sudo" "true"
    else
        config_set "enable_sudo" "false"
    fi
}

# Prompt ssh access if not set
prompt_ssh() {
    if [[ -n "$(config_get enable_ssh)" ]]; then
        return
    fi

    echo ""
    info "=== SSH Access ==="
    read -r -p "Enable ssh access to container? (y/N): " ssh_choice
    if [[ $ssh_choice =~ ^[Yy]$ ]]; then
        config_set "enable_ssh" "true"
        prompt_ssh_key
    else
        config_set "enable_ssh" "false"
    fi
}

prompt_ssh_key() {
    local ssh_dir="$HOME/.ssh"
    local pub_keys=()
    
    # Find all public keys
    if [[ -d "$ssh_dir" ]]; then
        while IFS= read -r key; do
            [[ -n "$key" ]] && pub_keys+=("$key")
        done < <(find "$ssh_dir" -maxdepth 1 -name "*.pub" -type f 2>/dev/null)
    fi

    
    if [[ ${#pub_keys[@]} -eq 0 ]]; then
        warn "No SSH public keys found in ~/.ssh/"
        config_set "ssh_key_path" ""
        return
    fi
    
    echo ""
    echo "Select SSH public key to mount:"
    for i in "${!pub_keys[@]}"; do
        echo "  $((i+1))) $(basename "${pub_keys[$i]}")"
    done
    echo "  $((${#pub_keys[@]}+1))) Custom path"
    echo "  $((${#pub_keys[@]}+2))) Skip (add manually later)"
    
    read -r -p "Choice [1]: " key_choice
    key_choice=${key_choice:-1}
    
    if [[ $key_choice -ge 1 && $key_choice -le ${#pub_keys[@]} ]]; then
        config_set "ssh_key_path" "${pub_keys[$((key_choice-1))]}"
    elif [[ $key_choice -eq $((${#pub_keys[@]}+1)) ]]; then
        read -r -p "Enter path to public key: " custom_key
        if [[ -f "$custom_key" ]]; then
            config_set "ssh_key_path" "$custom_key"
        else
            warn "File not found: $custom_key"
            config_set "ssh_key_path" ""
        fi
    else
        config_set "ssh_key_path" ""
    fi
}

# Prompt for ports if not set
prompt_ports() {
    local ports_var="$1"

    echo ""
    info "=== Port Forwarding ==="
    echo "Enter supplementary container ports to expose (space-separated, or Enter to skip):"
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

    # Get user's home directory and normalize it
    local user_home
    if [[ $IS_WINDOWS -eq 1 ]]; then
        user_home=$(normalize_path "$HOME")
    else
        user_home="$HOME"
    fi

    for choice in $choices; do
        case $choice in
            1)
                eval "$volumes_var+=('$user_home/.ssh:/home/\${USERNAME}/.ssh:ro')"
                ;;
            2)
                eval "$volumes_var+=('$user_home/.git-credentials:/home/\${USERNAME}/.git-credentials:ro')"
                ;;
            3)
                eval "$volumes_var+=('$user_home/.aws:/home/\${USERNAME}/.aws:ro')"
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
        # Normalize the host path portion if it looks like an absolute path
        if [[ "$vol" =~ ^([^:]+):(.+)$ ]]; then
            local host_part="${BASH_REMATCH[1]}"
            local container_part="${BASH_REMATCH[2]}"
            host_part=$(normalize_path "$host_part")
            vol="$host_part:$container_part"
        fi
        eval "$volumes_var+=('$vol')"
    done
}

# Commands
cmd_create() {
    config_init

    local name=""
    local project_host_path=""
    local ports=()
    local volumes=()
    local no_prompt=0

    # First two positional args
    if [[ $# -lt 1 ]]; then
        error "Usage: paul-envs.sh create <project-path> [options]"
    fi

    project_host_path=$(get_absolute_path "$1")
    config_set "project_host_path" "$project_host_path"
    shift 1

    # Parse flags
    while [[ $# -gt 0 ]]; do
        case $1 in
            --no-prompt)
                no_prompt=1
                shift
                ;;
            --name)
                validate_project_name "$2"
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
            --enable-wasm|--wasm)
              config_set "enable_wasm" "true"
              shift
              ;;
            --enable-ssh|--ssh)
              config_set "enable_ssh" "$2"
              shift
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
            --neovim)
                config_set "install_neovim" "true"
                shift
                ;;
            --starship)
                config_set "install_starship" "true"
                shift
                ;;
            --atuin)
                config_set "install_atuin" "true"
                shift
                ;;
            --mise)
                config_set "install_mise" "true"
                shift
                ;;
            --zellij)
                config_set "install_zellij" "true"
                shift
                ;;
            --jujutsu)
                config_set "install_jujutsu" "true"
                shift
                ;;
            --port)
                validate_port "$2"
                ports+=("$2")
                shift 2
                ;;
            --volume)
                # Normalize volume host paths
                local vol="$2"
                if [[ "$vol" =~ ^([^:]+):(.+)$ ]]; then
                    local host_part="${BASH_REMATCH[1]}"
                    local container_part="${BASH_REMATCH[2]}"
                    host_part=$(normalize_path "$host_part")
                    vol="$host_part:$container_part"
                fi
                volumes+=("$vol")
                shift 2
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done

    # Determine project name
    if [[ -z "$name" ]]; then
        name="$(basename "$(config_get project_host_path)")"
    fi

    name="$(sanitize_project_name "$name")"

    config_set "project_dest_path" "$name"
    check_inexistent_name "$name"

    # If --no-prompt, validate we have everything needed
    if [[ $no_prompt -eq 1 ]]; then
        # Set defaults for anything not specified
        if [[ -z "$(config_get shell)" ]]; then
            config_set "shell" "bash"
        fi
        if [[ -z "$(config_get install_node)" ]]; then
            config_set "install_node" "none"
        fi
        if [[ -z "$(config_get install_rust)" ]]; then
            config_set "install_rust" "none"
        fi
        if [[ -z "$(config_get install_python)" ]]; then
            config_set "install_python" "none"
        fi
        if [[ -z "$(config_get install_go)" ]]; then
            config_set "install_go" "none"
        fi
        if [[ -z "$(config_get enable_wasm)" ]]; then
            config_set "enable_wasm" "false"
        fi
        if [[ -z "$(config_get enable_ssh)" ]]; then
            config_set "enable_ssh" "false"
        fi
        if [[ -z "$(config_get enable_sudo)" ]]; then
            config_set "enable_sudo" "false"
        fi
        if [[ -z "$(config_get install_neovim)" ]]; then
            config_set "install_neovim" "false"
        fi
        if [[ -z "$(config_get install_starship)" ]]; then
            config_set "install_starship" "false"
        fi
        if [[ -z "$(config_get install_atuin)" ]]; then
            config_set "install_atuin" "false"
        fi
        if [[ -z "$(config_get install_mise)" ]]; then
            config_set "install_mise" "false"
        fi
        if [[ -z "$(config_get install_zellij)" ]]; then
            config_set "install_zellij" "false"
        fi
        if [[ -z "$(config_get install_jujutsu)" ]]; then
            config_set "install_jujutsu" "false"
        fi
        mise_check $no_prompt
    else
        # Interactive mode - prompt for missing values
        prompt_shell
        prompt_languages
        prompt_tools
        mise_check $no_prompt
        prompt_sudo
        prompt_ssh
        prompt_packages

        # Only prompt for ports if none were specified
        if [[ ${#ports[@]} -eq 0 ]]; then
            prompt_ports ports
        fi

        # Only prompt for volumes if none were specified
        if [[ ${#volumes[@]} -eq 0 ]]; then
            prompt_volumes volumes
        fi
    fi

    # Validate path exists or warn
    mkdir -p "$PROJECTS_DIR"

    local final_path
    final_path="$(config_get project_host_path)"

    if [[ ! -d "$final_path" && $no_prompt -eq 0 ]]; then
        warn "Warning: Path $final_path does not exist"
        read -p "Create config anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi

    # Generate config
    generate_project_compose "$name" "${ports[@]}" "VOLUMES_START" "${volumes[@]}"

    success "Created project '$name'"
    echo ""
    echo "Next steps:"
    echo "  1. Review/edit configuration:"
    echo "     - $(get_project_env "$name")"
    echo "     - $(get_project_compose "$name")"
    echo "  2. Put the \$HOME dotfiles you want to port in:"
    echo "     - $SCRIPT_DIR/configs/"
    echo "  3. Build the environment:"
    echo "     paul-envs.sh build $name"
    echo "  4. Run the environment:"
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
        echo "No project name given, listing projects..."
        echo ""
        cmd_list
        echo ""
        read -r -p "Enter project name to build: " name
        if [[ -z "$name" ]]; then
            error "No project name provided"
        fi
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
    shift 1 || true

    if [[ -z "$name" ]]; then
        echo "No project name given, listing projects..."
        echo ""
        cmd_list
        echo ""
        read -r -p "Enter project name to run: " name
        if [[ -z "$name" ]]; then
            error "No project name provided"
        fi
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

    # If additional arguments are provided, pass them to docker compose run
    # Otherwise, start an interactive shell (default behavior)
    if [[ $# -eq 0 ]]; then
        docker compose -f "$BASE_COMPOSE" -f "$compose_file" --env-file "$env_file" run --rm paulenv
    else
        docker compose -f "$BASE_COMPOSE" -f "$compose_file" --env-file "$env_file" run --rm paulenv "$@"
    fi
}

cmd_remove() {
    local name=$1

    if [[ -z "$name" ]]; then
        echo "No project name given, listing projects..."
        echo ""
        cmd_list
        echo ""
        read -r -p "Enter project name to build: " name
        if [[ -z "$name" ]]; then
            error "No project name provided"
        fi
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

cmd_version() {
    echo "paul-envs.sh version 0.1.0"
    echo "Bash version: $BASH_VERSION"
    echo "Docker version: $(docker --version 2>/dev/null || echo 'not installed')"
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
    version|--version|-v)
        cmd_version
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
  --packages "PKG1 PKG2"   Additional Ubuntu packages (prompted if not specified)
  --port PORT              Expose container port (prompted if not specified, can be repeated)
  --volume HOST:CONT[:ro]  Mount volume (prompted if not specified, can be repeated)

Windows/Git Bash Notes:
  - Paths are automatically converted (C:\Users\... -> /c/Users/...)
  - UID/GID default to 1000 on Windows (Docker Desktop requirement)
  - Use forward slashes or let the script normalize paths for you

Interactive Mode (default):
  paul-envs.sh create ~/projects/myapp
  # Will prompt for all unspecified options

Non-Interactive Mode:
  paul-envs.sh create ~/projects/myapp --no-prompt --shell bash --nodejs latest

Mixed Mode (some flags + prompts):
  paul-envs.sh create ~/projects/myapp --nodejs 20.10.0 --rust latest --mise
  # Will prompt for shell, sudo, packages, ports, and volumes

Full Configuration Example:
  paul-envs.sh create ~/work/api \\
    --name myApp \\
    --shell zsh \\
    --nodejs 20.10.0 \\
    --rust latest \\
    --python 3.12.0 \\
    --go latest \\
    --mise \\
    --neovim \\
    --starship \\
    --zellij \\
    --jujutsu \\
    --enable-ssh \\
    --enable-sudo \\
    --git-name "John Doe" \\
    --git-email "john@example.com" \\
    --packages "ripgrep fzf" \\
    --port 3000 \\
    --port 5432 \\
    --volume ~/.git-credentials:/home/dev/.git-credentials:ro

Configuration:
  Base compose: $BASE_COMPOSE
  Projects directory: $PROJECTS_DIR
EOF
        ;;
esac
