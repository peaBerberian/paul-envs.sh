# paul-envs

`paul-envs` is a dev container manager to let me easily work on multiple large
projects with rapidly changing dependencies in isolated and minimal
environments.

<video src="https://github.com/user-attachments/assets/0eb8bbb8-5ad4-4c8d-8d80-f92fbb0072c4"></video>

_Video: Creating and running a project with `paul-envs`. Here even development
itself can take place inside the container._

## What's this

`paul-envs` is both a wrapper over the `docker compose` tool and a configuration
generator for it.

Each of the created containers is similar in a way to [dev
containers](https://containers.dev/) in that they are targeted for development
usages but mine is optimized for multi-projects workflows and CLI-heavy usage.

Key features:

-  **Ephemeral by default**: Only project files and key directories (e.g.
   terminal history, installed tools' stored data, editor plugins...) are
   persisted. Everything else resets on exit, keeping the environment clean.

-  **Multi-projects**: Multiple images can be handled, each being linked to a
   project directory on your host.

-  **Minimal base**: The containers are just Ubuntu LTS and your chosen CLI
   tools. This means no unnecessary package and a very common - thus tested -
   base.

-  **Dev-oriented**: Possibility to opt-in to the installation of many popular
   CLI tools (`neovim`, `atuin`, `mise`, `jujutsu`, `zellij`...) as well as
   many language toolkits (Node.js + npm, Rust + cargo, go, python + pip + venv
   and WebAssembly tools like binaryen).

-  **optional SSH**: You can opt-in to ssh access from the host (e.g. for
   relying on your host's GUI editor), just like "devcontainers".

-  **Shared caches**: cache directories are shared across all projects to avoid
   redundant downloads.

-  **Fast setup**: Single shared `Dockerfile` means new project containers
   build quickly.

-  **Easy to use**: I made it compatible with MacOS, Linux and Windows, with
   automatic x86_64 or arm64 container creation depending on the host.

   The `paul-envs` binary also guides you when you call any command without
   argument.


## Comparison with other similar tools

Regarding **alternatives**, `paul-envs` fit in a sweet spot for me:

**vs. dev containers:**
Dev containers include IDE integration, is generally plug-and-play, and has a
rich ecosystem. `paul-envs` is editor-agnostic, CLI-specialized, handle multiple
projects directly and is simpler conceptually.

**vs. Devbox:**
Devbox has deterministic and reproducible environments thanks to `nix`, yet
doesn't provide complete isolation - e.g. your own `$HOME` directory can still
be updated by your project's scripts.
`paul-envs` rely on full container isolation at the cost of less
reproducibility, it also relies on a familiar Ubuntu LTS instead of `nix`.

**vs. docker compose:**
`paul-envs` wraps `docker compose` calls and add multi-project management and
good defaults and setup for a CLI-based development environment.
It can be seen as a "convenience layer" on top of `docker compose`.

**vs. nix-shell / direnv:**
These manipulate your environment (PATH, env vars) but don't provide container
isolation (same issue than with `devbox`).

## Why creating this

I often have to switch between projects at work.

Some of those projects are very fast-moving JS projects with a lot of
dependencies churning, and new tools being added very frequently. Some of those
update system files (e.g. the `mkcert` tool) without my explicit agreement.

Moreover, most of those developers also have very similar environments between
each other, which is sadly not close to mine, so I encounter a lot of issues
that they never encounter. As those are huge projects and not my main focus, the
right action which would be to just fix those issues is very time-consuming.

I thus decided to rely on containers for developing on those projects to
protect my own system from unwanted changes and to provide a more "barebone"
and popular environment (ubuntu LTS, adding only my current developing tools to
it).
As my setup is only terminal-based (neovim, CLI tools), this can be done
easily.

At first I was just relying on `systemd-nspawn`, as this is a tool that I knew.
But by using it for this, I thought that having a base with an ephemeral
"overlay" and a few mounted "volumes" from the host (for the source code, caches
and some minimal controlled state such as shell history) was the most flexible
solution for my setup, so I ended up with a more complex `Dockerfile`
and `compose.yaml` file instead and I now rely on a software compatible to
those.

In the end I spent some efforts making sure those files are minimal, portable
and optimized enough to efficienty be relied on for multi-projects setups, each
with its own container. For example, the order of instruction in the
`Dockerfile` have been carefully thought out to perform tasks from the
less-likely to change to the most likely for efficient caching and some
package-side caching (e.g. `yarn`, `npm` etc.) is shared between all containers
through persistent volumes.

## How to run it

First, download the latest binary [in our releases
page](https://github.com/peaBerberian/paul-envs/releases/latest).

Running that executable without any argument will list all available operations
and corresponding flags.

### 1. Create a new container's config

The idea is to create a separate container for each project (that will rely on a
same base with variations).

This container first need to be configured to point to your project and have the
right arguments (e.g. the right tools and git configuration). This is done
through the `paul-envs create` "command".

First ensure the target project is present locally in your host, then run:
```sh
paul-envs create <path/to/your/project>
```

Optionally, you may add a lot of flags to better configure that container.
Here's an example of a real-life usage:
```sh
# Will create a container named `myapp` with a default `zsh` shell and many
# configurations. Also mount your `.git-credentials` readonly to the container.
paul-envs create myapp ~/work/api \
  --name myProject \
  --shell zsh \
  --node-version 22.11.0 \
  --git-name "John Doe" \
  --git-email "john@example.com" \
  --port 8000 \
  --port 5432 \
  --package fzf \
  --package ripgrep \
  --volume ~/.git-credentials:/home/dev/.git-credentials:ro
```

Without the corresponding flags, prompts will be proposed by `paul-envs` for
important parameters (choosen shell, wanted pre-mounted volumes etc.).

What this step does is just to create both a `yaml` and a `.env` file containing
your container's configuration in your application data directory (advertised
after the command succeeds). It doesn't build anything yet.

### 2. Build the container

The previous file created both a "compose" and "env" file - basically
configuration to define the container we want to build.

This step relies on `docker compose`, which you should have locally installed.

To build a container, just run the `paul-envs build <NAME>` command.
For example, with a container named `myApp`, you would just do:
```sh
paul-envs build myApp
```

This will take some time as the initialization of the container is going on:
packages are loaded, tools are set-up etc.

### 3. Run the container

Now that the container is built. It can be run at any time, with the
`paul-envs run` command.
For example, with a container named `myApp`, you would do:
```sh
paul-envs run myApp
```

You will directly switch to the mounted project directory inside that container.

You can go out of that container at any time (e.g. by calling `exit` or hitting
`Ctrl+D`), as you exit that container, everything that is not part of the
"persisted volume" (see `What gets preserved vs. ephemeral` chapter) is reset to
the state it was at build-time.

### Other commands

`paul-envs` also proposes multiple other commands:
```sh
# List all created configurations, built or not
paul-envs list

# Remove the configuration file and container data for the `myApp` project
paul-envs remove myApp

# Get version information
paul-envs version

# Start an interactive session
paul-envs interactive

# Display global help
paul-envs help

# Uninstall paul-envs completely from your system (remove all projects, config etc.)
paul-envs clean
```

### Note: The dotfiles directory

The "dotfiles directory" is a special location that will be merged with the home
directory of an image on a `build` command.

As such you can put a `.bashrc` directly in there, and the config for
the tools you planned to install (e.g. the `starship` configuration file:
`starship.toml`, a `nvim` directory for `neovim` etc.):
```
dotfiles_dir/
├── .bashrc
└── .config/
    ├── starship.toml (config for the starship tool)
    └── nvim/
        └── ... (your neovim config)
```

All its content will be copied as is unmodified, with two exceptions:

1.  shell files  (`.bashrc`, `.zshrc` and/or `.config/fish/config.fish` files)
    may still be updated after being copied in the dockerfile to redirect their
    history to ensure history persistence.

2.  The git config file (generally `~/.gitconfig`) may also be updated after
    being copied in the container to set the name and e-mail information you
    configured in your env file.

Because those are the only exceptions, if you plan to overwrite one of those
shell files, you will need to add the tool initialization commands in them
yourself (e.g. `eval "$(starship init bash)"` for initializing `starship` in the
bash shell in your `.bashrc`).
If you're not overwriting those files however, the default provided one will
already contain the initialization code for all the tools explicitely listed in
the dockerfile.

The job of copying the dotfiles directory's content is taken by the
`Dockerfile`. Meaning that you'll profit from this even if you're not relying on
`docker compose` or `paul-envs`.

If you don't go through `paul-envs`, you will have to set the `DOTFILES_DIR` env
variable yourself so it points to the dotfiles directory you defined yourself.
When relying on `paul-envs`, a directory will be created for you.

## What gets preserved vs. ephemeral

When working inside the container, here's what you can expect to be either
"preserved" (changes will stay from container to container) or "ephemeral" (it
will be removed when the container is exited).

- **Preserved**: the mounted project directory (`~/projects/<NAME>`), the
  "cache" directory (mounted as `~/.container-cache`) and the "local" directory
  (mounted as `~/.container-local`) - see the "persisted volumes" chapter for
  those last two.

- **Ephemeral**: All other changes (further installed global packages, global
  system configurations etc.)

## TODO:

- help flag per commands
- Add "init bash / zsh /fish" commands to simplify auto-completion setups
- no-prompt flags for clean, remove...
- `update` command?
- `kill` command?
- `up` command?
- rootless support
- Add `kakoune` and `helix` as potential in-container editors
- less gh-action scripts, more shell scripts
- Kill containers on same image on build?
- Reference counted container instead of master/slaves
- Does cache pruning in `clean` actually does anything?
- ci tests for clean command
