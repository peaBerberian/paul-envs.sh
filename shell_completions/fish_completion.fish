# Fish completion for paul-envs

# Helper function to get container names from paul-envs ls
function __paul_envs_containers
    paul-envs list --names 2>/dev/null
end

# Main commands
complete -c paul-envs -f -n __fish_use_subcommand -a interactive -d 'Start interactive mode'
complete -c paul-envs -f -n __fish_use_subcommand -a create -d 'Create a container configuration'
complete -c paul-envs -f -n __fish_use_subcommand -a list -d 'List all available containers'
complete -c paul-envs -f -n __fish_use_subcommand -a build -d 'Build a container'
complete -c paul-envs -f -n __fish_use_subcommand -a run -d 'Start a container'
complete -c paul-envs -f -n __fish_use_subcommand -a remove -d 'Remove a container'
complete -c paul-envs -f -n __fish_use_subcommand -a help -d 'Show help'
complete -c paul-envs -f -n __fish_use_subcommand -a version -d 'Show version'
complete -c paul-envs -f -n __fish_use_subcommand -a clean -d 'Remove all stored paul-envs data from your computer'

# Create command options
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l name -d "Specific a container name" -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l uid -d 'Host UID' -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l gid -d 'Host GID' -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l username -d 'Container username' -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l shell -d 'User shell' -xa 'bash zsh fish'
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l nodejs -d 'Node.js installation' -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l rust -d 'Rust installation' -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l python -d 'Python installation' -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l go -d 'Go installation' -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l git-name -d 'Git author name' -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l git-email -d 'Git author email' -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l package -d 'Additional Ubuntu package' -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l enable-ssh -d "Enable ssh access" -f
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l enable-sudo -d "Enable sudo access (password: \"dev\")" -f
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l neovim -d "Install Neovim" -f
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l starship -d "Install Starship" -f
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l atuin -d "Install Atuin" -f
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l mise -d "Install Mise" -f
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l zellij -d "Install Zellij" -f
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l jujutsu -d "Install Jujutsu" -f
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l port -d 'Expose port' -x
complete -c paul-envs -n "__fish_seen_subcommand_from create" -l volume -d 'Add volume' -r

complete -c paul-envs -n "__fish_seen_subcommand_from list" -l names -d "Only display names" -f

# Container name completion for build, run, remove
complete -c paul-envs -f -n "__fish_seen_subcommand_from build" -a '(__paul_envs_containers)'
complete -c paul-envs -f -n "__fish_seen_subcommand_from run" -a '(__paul_envs_containers)'
complete -c paul-envs -f -n "__fish_seen_subcommand_from remove" -a '(__paul_envs_containers)'
