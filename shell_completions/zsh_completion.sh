#compdef paul-envs.sh

_paulenvs() {
    local -a commands
    commands=(
        'create:Create a new development container'
        'list:List all containers'
        'build:Build an container'
        'run:Run commands in an container'
        'remove:Remove an container'
    )

    _arguments -C \
        '1: :->command' \
        '*: :->args' && return 0

    case $state in
        command)
            _describe 'command' commands
            ;;
    esac
}

_paulenvs "$@"
