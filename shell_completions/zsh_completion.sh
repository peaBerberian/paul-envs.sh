#compdef paul-envs

_paulenvs() {
    local -a commands
    commands=(
        'create:Create a container configuration'
        'list:List all available containers'
        'build:Build a container'
        'run:Start a container'
        'remove:Remove a container'
    )

    # Get list of existing containers from paul-envs ls
    local -a containers
    containers=(${(f)"$(paul-envs ls 2>/dev/null | grep -E '^\s+-\s+' | sed 's/^\s*-\s*//')"})


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
                        '2:project path:_directories' \
                        '--name[Specify container name]:name:' \
                        '--uid[Host UID]:uid:($(id -u))' \
                        '--gid[Host GID]:gid:($(id -g))' \
                        '--username[Container username]:username:' \
                        '--shell[User shell]:shell:(bash zsh fish)' \
                        '--nodejs[Node.js installation]:version:' \
                        '--rust[Rust installation]:version:' \
                        '--python[Python installation]:version:' \
                        '--go[Go installation]:version:' \
                        '--git-name[Git author name]:name:' \
                        '--git-email[Git author email]:email:' \
                        '--packages[Additional packages]:packages:' \
                        '--enable-ssh[Enable ssh access]' \
                        '--enable-sudo[Enable sudo access (password: \"dev\")]' \
                        '--neovim[Install latest Neovim]' \
                        '--starship[Install latest Starship]' \
                        '--atuin[Install latest Atuin]' \
                        '--mise[Install latest Mise]' \
                        '--zellij[Install latest Zellij]' \
                        '--jujutsu[Install latest Jujutsu]' \
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
