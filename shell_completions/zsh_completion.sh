#compdef paul-envs.sh

_paulenvs() {
    local -a commands
    commands=(
        'create:Create a container configuration'
        'list:List all available containers'
        'build:Build a container'
        'run:Start a container'
        'remove:Remove a container'
    )

    # Get list of existing containers from paul-envs.sh ls
    local -a containers
    containers=(${(f)"$(paul-envs.sh ls 2>/dev/null | grep -E '^\s+-\s+' | sed 's/^\s*-\s*//')"})


    _arguments -C \
        '1: :->command' \
        '*: :->args' && return 0

    case $state in
        command)
            _describe 'command' commands
            ;;
        args)
            case ${words[2]} in
                create)
                    _arguments \
                        '2:project name:' \
                        '3:project path:_directories' \
                        '--uid[Host UID]:uid:($(id -u))' \
                        '--gid[Host GID]:gid:($(id -g))' \
                        '--username[Container username]:username:' \
                        '--shell[User shell]:shell:(bash zsh fish)' \
                        '--node-version[Node.js version]:version:' \
                        '--git-name[Git author name]:name:' \
                        '--git-email[Git author email]:email:' \
                        '--packages[Additional packages]:packages:' \
                        '--no-neovim[Skip Neovim installation]' \
                        '--no-starship[Skip Starship installation]' \
                        '--no-atuin[Skip Atuin installation]' \
                        '--no-mise[Skip Mise installation]' \
                        '--no-zellij[Skip Zellij installation]' \
                        '*--port[Expose port]:port:' \
                        '*--volume[Add volume]:volume:_files'
                    ;;
                build)
                    _arguments \
                        "2:container name:(${containers[@]})"
                    ;;
                run)
                    _arguments \
                        "2:container name:(${containers[@]})" \
                        '*:command:'
                    ;;
                remove)
                    _arguments \
                        "2:container name:(${containers[@]})"
                    ;;
                list)
                    # No additional arguments
                    ;;
            esac
            ;;
    esac
}

_paulenvs "$@"
