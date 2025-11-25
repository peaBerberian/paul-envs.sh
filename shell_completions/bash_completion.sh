_paulenvs()
{
    local cur prev opts create_opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main commands
    local commands="create list build run remove version interactive help clean"

    # Options for create command
    local create_flags="--name --uid --gid --username --shell --nodejs --rust --python --go --git-name --git-email --package --enable-ssh --enable-sudo --neovim --starship --atuin --mise --zellij --jujutsu --port --volume"

    # Get list of existing containers from paul-envs ls
    _get_containers() {
        paul-envs ls 2>/dev/null | grep -E '^\s+-\s+' | sed 's/^\s*-\s*//'
    }

    # First argument (command)
    if [[ $COMP_CWORD -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi

    local command="${COMP_WORDS[1]}"

    case "${command}" in
        interactive)
            # No further completion
            return 0
            ;;
        create)
            case "${prev}" in
                --uid|--gid)
                    # Could suggest current UID/GID
                    COMPREPLY=( $(compgen -W "$(id -u) $(id -g)" -- ${cur}) )
                    return 0
                    ;;
                --username|--git-name|--git-email|--package|--nodejs|--rust|--python|--go|--port)
                    # Let user type freely
                    COMPREPLY=()
                    return 0
                    ;;
                --shell)
                    COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
                    return 0
                    ;;
                --volume)
                    # Complete file paths
                    COMPREPLY=( $(compgen -f -- ${cur}) )
                    return 0
                    ;;
                create)
                    # After 'create', expect project name (no completion)
                    COMPREPLY=()
                    return 0
                    ;;
                *)
                    # If previous was a project name, suggest path completion
                    # Otherwise suggest flags
                    if [[ $COMP_CWORD -eq 3 ]]; then
                        # Third argument: project path
                        COMPREPLY=( $(compgen -d -- ${cur}) )
                    else
                        # Suggest create flags
                        COMPREPLY=( $(compgen -W "${create_flags}" -- ${cur}) )
                    fi
                    return 0
                    ;;
            esac
            ;;
        build|run|remove)
            # Complete with container names
            if [[ $COMP_CWORD -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$(_get_containers)" -- ${cur}) )
            fi
            return 0
            ;;
        list|help|version|clean)
            # No further completion
            return 0
            ;;
    esac
}

complete -F _paulenvs paul-envs
