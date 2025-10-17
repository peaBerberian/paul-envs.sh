#!/bin/bash
set -e

# Locate the base compose file, should be in the same directory than this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_COMPOSE="$SCRIPT_DIR/compose.yaml"

# Directory where projects' yaml and env files will be created
PROJECTS_DIR="$SCRIPT_DIR/projects"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
error() {
  echo -e "${RED}Error: $1${NC}" >&2;
  exit 1;
}
success() {
  echo -e "${GREEN}$1${NC}";
}
warn() {
  echo -e "${YELLOW}$1${NC}";
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

validate_node_version() {
    local version=$1
    # Allow "latest" or semantic versioning patterns
    if [[ "$version" != "latest" && ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        error "Invalid node version '$version'. Must be 'latest' or semantic version (e.g., 20.10.0)"
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

# Generate project compose file
# Usage: generate_project_compose name config_array ports_array volumes_array
generate_project_compose() {
    local name=$1
    local -n gen_cfg=$2
    local -n gen_ports=$3
    local -n gen_volumes=$4

    local compose_file=$(get_project_compose "$name")
    local env_file=$(get_project_env "$name")

    if [[ -f "$compose_file" || -f "$env_file" ]]; then
        error "Project '$name' already exists\nHint: Use 'paul-envs.sh list' to see all projects or 'paul-envs.sh remove $name' to delete it"
    fi

    # Sanitize all user inputs
    local safe_git_name=$(light_sanitize "${gen_cfg[git_name]}")
    local safe_git_email=$(light_sanitize "${gen_cfg[git_email]}")
    local safe_packages=$(light_sanitize "${gen_cfg[packages]}")

    mkdir -p "$(dirname "$compose_file")"

		# Generate .env file
    cat >> "$env_file" <<EOF
HOST_UID="${gen_cfg[host_uid]}"
HOST_GID="${gen_cfg[host_gid]}"
USERNAME="${gen_cfg[username]}"
USER_SHELL="${gen_cfg[shell]}"
NODE_VERSION="${gen_cfg[node_version]}"
PROJECT_PATH="${gen_cfg[project_path]}"
SUPPLEMENTARY_PACKAGES="$safe_packages"
INSTALL_NEOVIM="${gen_cfg[install_neovim]}"
INSTALL_STARSHIP="${gen_cfg[install_starship]}"
INSTALL_ATUIN="${gen_cfg[install_atuin]}"
INSTALL_MISE="${gen_cfg[install_mise]}"
EOF

    # Add git args if provided
    if [[ -n "$safe_git_name" ]]; then
        echo "GIT_AUTHOR_NAME=\"$safe_git_name\"" >> "$env_file"
    fi
    if [[ -n "$safe_git_email" ]]; then
        echo "GIT_AUTHOR_EMAIL=\"$safe_git_email\"" >> "$env_file"
    fi

    # Generate YAML
    cat >> "$compose_file" <<EOF
services:
  paulenv:
    build:
EOF

    # Add ports if specified
    if [[ ${#gen_ports[@]} -gt 0 ]]; then
        echo "    ports:" >> "$compose_file"
        for port in "${gen_ports[@]}"; do
            echo "      - \"$port:$port\"" >> "$compose_file"
        done
    fi

    # Add volumes if wanted
    cat >> "$compose_file" <<EOF
    volumes:
EOF
    # TODO: some kind of pre-validation?
    for vol in "${gen_volumes[@]}"; do
        echo "      - $vol" >> "$compose_file"
    done

    echo "" >> "$compose_file"
}

# Prompt for common credential mounts
prompt_for_credentials() {
    local username=$1
    local -n result_volumes=$2

    echo ""
    echo "Mount common credentials/configs? (space-separated numbers, or Enter to skip)"
    echo "  1) SSH keys (~/.ssh)"
    echo "  2) Git credentials (~/.git-credentials)"
    echo "  3) AWS credentials (~/.aws)"
    echo "  4) Custom CA certificates (/etc/ssl/certs/custom-ca.crt)"
    read -p "Choice [none]: " choices

    if [[ -z "$choices" ]]; then
        return
    fi

    for choice in $choices; do
        case $choice in
            1)
                result_volumes+=("~/.ssh:/home/\${USERNAME}/.ssh:ro")
                ;;
            2)
                result_volumes+=("~/.git-credentials:/home/\${USERNAME}/.git-credentials:ro")
                ;;
            3)
                result_volumes+=("~/.aws:/home/\${USERNAME}/.aws:ro")
                ;;
            4)
                result_volumes+=("/etc/ssl/certs/custom-ca.crt:/usr/local/share/ca-certificates/custom-ca.crt:ro")
                ;;
            *)
                warn "Unknown choice: $choice (skipped)"
                ;;
        esac
    done
}

# Commands
cmd_create() {
    declare -A config=(
        [host_uid]=$(id -u)
        [host_gid]=$(id -g)
        [username]="dev"
        [shell]=""
        [node_version]="latest"
        [git_name]=""
        [git_email]=""
        [packages]=""
        [install_neovim]="true"
        [install_starship]="true"
        [install_atuin]="true"
        [install_mise]="true"
    )

    local name=""
    local project_path=""
    local ports=()
    local volumes=()

    # First two positional args
    if [[ $# -lt 2 ]]; then
        error "Usage: paul-envs.sh create <name> <project-path> [options]"
    fi

    name=$1
    project_path=$2
    shift 2

    # Validate project name early
    validate_project_name "$name"

    # Parse flags
    while [[ $# -gt 0 ]]; do
        case $1 in
            --uid)
                validate_uid_gid "$2" "UID"
                config[host_uid]=$2
                shift 2
                ;;
            --gid)
                validate_uid_gid "$2" "GID"
                config[host_gid]=$2
                shift 2
                ;;
            --username)
                validate_username "$2"
                config[username]=$2
                shift 2
                ;;
            --shell)
                validate_shell "$2"
                config[shell]=$2
                shift 2
                ;;
            --node-version)
                validate_node_version "$2"
                config[node_version]=$2
                shift 2
                ;;
            --git-name)
                validate_git_name "$2"
                config[git_name]=$2
                shift 2
                ;;
            --git-email)
                validate_git_email "$2"
                config[git_email]=$2
                shift 2
                ;;
            --packages)
                validate_apt_package_names "$2"
                config[packages]=$2
                shift 2
                ;;
            --no-neovim)
                config[install_neovim]="false"
                shift
                ;;
            --no-starship)
                config[install_starship]="false"
                shift
                ;;
            --no-atuin)
                config[install_atuin]="false"
                shift
                ;;
            --no-mise)
                config[install_mise]="false"
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

    # Ask for shell if not provided
    if [[ -z "${config[shell]}" ]]; then
        echo "Select shell:"
        echo "  1) bash (default)"
        echo "  2) zsh"
        echo "  3) fish"
        read -p "Choice [1]: " shell_choice
        case ${shell_choice:-1} in
            1) config[shell]="bash" ;;
            2) config[shell]="zsh" ;;
            3) config[shell]="fish" ;;
            *) config[shell]="bash" ;;
        esac
    fi

    # Prompt for common credentials if no --volume flags were used
    if [[ ${#volumes[@]} -eq 0 ]]; then
        prompt_for_credentials "${config[username]}" volumes
    fi

    # Validate and expand project path
    mkdir -p "$PROJECTS_DIR"

    # Safe tilde expansion without eval
    if [[ "$project_path" == "~"* ]]; then
        project_path="${HOME}${project_path:1}"
    fi

    # Resolve to absolute path if relative
    if [[ "$project_path" != /* ]]; then
        project_path="$(pwd)/$project_path"
    fi

    config[project_path]=$project_path

    if [[ ! -d "$project_path" ]]; then
        warn "Warning: Path $project_path does not exist"
        read -p "Create config anyway? (y/N) " -n 1 -r
        echo
        [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
    fi

    # Generate config
    generate_project_compose "$name" config ports volumes

    success "Created project '$name'"
}

cmd_list() {
    check_base_compose
    if [[ ! -d "$PROJECTS_DIR" ]]; then
        echo "No project created yet"
        echo "Hint: Create one with 'paul-envs.sh create <name> <path>'"
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
        echo "Hint: Create one with 'paul-envs.sh create <name> <path>'"
    fi
}

cmd_build() {
    check_base_compose
    local name=$1

    if [[ -z "$name" ]]; then
        error "Usage: paul-envs.sh build <name>\nHint: Use 'paul-envs.sh list' to see available projects"
    fi

    validate_project_name "$name"

    local compose_file=$(get_project_compose "$name")
    local env_file=$(get_project_env "$name")
    if [[ ! -f "$compose_file" || ! -f "$env_file" ]]; then
        error "Project '$name' not found\nHint: Use 'paul-envs.sh list' to see available projects or 'paul-envs.sh create' to make a new one"
    fi

    # Ensure shared cache volume exists
    docker volume create paulenv-shared-cache 2>/dev/null || true

    export COMPOSE_PROJECT_NAME="$name"
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
        error "Usage: paul-envs.sh run <name> [command]\nHint: Use 'paul-envs.sh list' to see available projects"
    fi

    validate_project_name "$name"

    local compose_file=$(get_project_compose "$name")
    local env_file=$(get_project_env "$name")
    if [[ ! -f "$compose_file" || ! -f "$env_file" ]]; then
        error "Project '$name' not found\nHint: Use 'paul-envs.sh list' to see available projects"
    fi

    export COMPOSE_PROJECT_NAME="$name"
    docker compose -f "$BASE_COMPOSE" -f "$compose_file" --env-file "$env_file" run --rm paulenv "$@"
}

cmd_remove() {
    local name=$1

    if [[ -z "$name" ]]; then
        error "Usage: paul-envs.sh remove <name>\nHint: Use 'paul-envs.sh list' to see available projects"
    fi

    validate_project_name "$name"

    local project_dir=$(get_project_dir "$name")
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
  paul-envs.sh create <name> <path> [options]
  paul-envs.sh list
  paul-envs.sh build <name>
  paul-envs.sh run <name> [command]
  paul-envs.sh remove <name>

Options for create:
  --uid UID                    Host UID (default: current user)
  --gid GID                    Host GID (default: current group)
  --username NAME              Container username (default: dev)
  --shell SHELL                User shell: bash|zsh|fish (prompted if not set)
  --node-version VERSION       Node.js version (default: latest)
  --git-name NAME              Git author name (optional)
  --git-email EMAIL            Git author email (optional)
  --packages "PKG1 PKG2"       Additional Ubuntu packages (optional)
  --no-neovim                  Don't install Neovim
  --no-starship                Don't install Starship prompt
  --no-atuin                   Don't install Atuin shell history
  --no-mise                    Don't install Mise tool manager
  --port PORT                  Expose port (can be repeated)
  --volume HOST:CONTAINER:ro   Add volume (can be repeated)

Examples:
  # Create a project (will ask for shell and credentials)
  paul-envs.sh create myapp ~/projects/myapp

  # Create with all options
  paul-envs.sh create myapp ~/work/api \\
    --shell zsh \\
    --node-version 20.10.0 \\
    --git-name "John Doe" \\
    --git-email "john@example.com" \\
    --packages "ripgrep fzf" \\
    --no-atuin \\
    --port 3000 \\
    --port 5432 \\
    --volume ~/.git-credentials:/home/dev/.git-credentials:ro

  # Then build it
  paul-envs.sh build myapp

  # Then run it
  paul-envs.sh run myapp

  # Or run a specific command
  paul-envs.sh run myapp cd app && npm run test

  # Manage projects
  paul-envs.sh list
  paul-envs.sh remove myapp

Configuration:
  Base compose: $BASE_COMPOSE
  Projects: Stored in directory you specify
EOF
        ;;
esac
