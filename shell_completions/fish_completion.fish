# Fish completion for paul-envs.sh

# Helper function to get container names from paul-envs.sh ls
function __paul_envs_containers
    paul-envs.sh ls 2>/dev/null | grep -E '^\s+-\s+' | sed 's/^\s*-\s*//'
end

# Main commands
complete -c paul-envs.sh -f -n __fish_use_subcommand -a create -d 'Create a container configuration'
complete -c paul-envs.sh -f -n __fish_use_subcommand -a list -d 'List all available containers'
complete -c paul-envs.sh -f -n __fish_use_subcommand -a build -d 'Build a container'
complete -c paul-envs.sh -f -n __fish_use_subcommand -a run -d 'Start a container'
complete -c paul-envs.sh -f -n __fish_use_subcommand -a remove -d 'Remove a container'

# Create command options
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l uid -d 'Host UID' -x
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l gid -d 'Host GID' -x
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l username -d 'Container username' -x
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l shell -d 'User shell' -xa 'bash zsh fish'
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l nodejs -d 'Node.js installation' -x
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l rust -d 'Rust installation' -x
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l git-name -d 'Git author name' -x
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l git-email -d 'Git author email' -x
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l packages -d 'Additional Ubuntu packages' -x
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l no-neovim -d "Don't install Neovim" -f
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l no-starship -d "Don't install Starship" -f
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l no-atuin -d "Don't install Atuin" -f
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l no-mise -d "Don't install Mise" -f
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l no-zellij -d "Don't install Zellij" -f
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l port -d 'Expose port' -x
complete -c paul-envs.sh -n "__fish_seen_subcommand_from create" -l volume -d 'Add volume' -r

# Container name completion for build, run, remove
complete -c paul-envs.sh -f -n "__fish_seen_subcommand_from build" -a '(__paul_envs_containers)'
complete -c paul-envs.sh -f -n "__fish_seen_subcommand_from run" -a '(__paul_envs_containers)'
complete -c paul-envs.sh -f -n "__fish_seen_subcommand_from remove" -a '(__paul_envs_containers)'
