# paul-envs.sh

`paul-envs.sh` allows me to manage development containers so I can easily work
on multiple large projects with rapidly changing dependencies in isolated and
minimal containers.

## What's this

`paul-envs` is built for me and other developers with similar needs: working on
multiple projects simultaneously, relying on only terminal-based workflows
(vim/neovim/kakoune/helix..., shell and CLI tools), and wanting protection from
projects that might modify system state or require conflicting dependencies.
This is done by providing a sane, mostly-ephemeral and minimal environment.

It is both a wrapper over the `docker compose` tool and a configuration
generator for it.
Each of the created containers is similar in a way to [dev
containers](https://containers.dev/) in that they are targeted for development
usages but mine is optimized for CLI-only, multi-projects workflows.

Key features:

-  **Ephemeral by default**: Only project files and key directories (e.g.
   terminal history, installed tools' stored data, editor plugins...) are
   persisted. Everything else resets on exit, keeping the environment clean.

-  **Shared caches**: npm/yarn caches are shared across all projects to avoid
   redundant downloads.

-  **Minimal base**: The containers are just Ubuntu LTS and your chosen CLI
   tools. There's no GUI, no unnecessary packages.

-  **Fast setup**: Single shared `Dockerfile` means new project containers
   build quickly.


## Comparison with other similar tools

Regarding **alternatives**, `paul-envs` fit in a sweet spot for me:

**vs. dev containers:**
Dev containers include IDE integration and a rich ecosystem. `paul-envs` is
editor-agnostic, CLI-specialized, handle multiple projects directly and is much
simpler conceptually (it's just a minimal Ubuntu LTS image with only CLI tools
wanted and configured on top) if all that is needed is a terminal-based
workflow.

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

## Quick Start

1. Clone this repository, `cd` to it, and ensure `docker compose` is installed
   locally and accessible in path.

2. Run `./paul-envs.sh create <path/to/your/project> --name myApp`.

   This will just create a compose and env file in a new `projects/` directory
   with the right preset properties and name the project `myApp`. If no
   `--name` flag is provided, the choosen name will be the name of the project's
   directory.

3. Optionally, put the "dotfiles" that you want to retrieve in the container's
   home directory in `configs`. They will be copied to the container when it is
   build (next step).

   Note that you shouldn't put your credentials/secrets in there (`~/.ssh`,
   `~/.aws`, `~/.git-credentials` etc.) as those could have issues being
   copied (due to restrictive permissions).

   If you want to copy some of those, see `./paul-envs.sh create` flags.

4. Run `./paul-envs.sh build myApp`.

   It will build the container through the right `docker compose build`
   invokation and initialize persistent volumes.

5. Then run the container each time you want to work on the project:
   `./paul-envs.sh run myApp`.

   The mounted project is available in that container at `~/projects/myApp`.

   The project, caches (npm/yarn caches etc.) and the pre-installed tools'
   storage (shell history, `mise` data etc.) are persisted, everything else is
   automically removed when that container is exited.

   Each new run thus start from a relatively clean and simple state.

## Why creating this

I often have to switch between projects at work.

Some of those projects are very fast-moving JS projects with a lot of
dependencies churning, and new tools being added very frequently. Some of those
update system files (e.g. the `mkcert` tool) without my explicit agreement.

Moreover, most of those developers also have very similar environments between
each other, which is sadly not close to mine, so I encounter a lot of issues
that they never encounter. As those are huge projects and not my main focus, the
right action which would be to just fix those issues is very time-consuming.

I thus decided to rely on a container for developing on those projects to
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

First you need to clone this repository and make it your current working
directory:
```sh
git clone https://github.com/peaBerberian/paul-envs.sh.git
cd paul-envs.sh
```

Running `./paul-envs.sh` without any argument will list all available operations
and corresponding flags:
```sh
./paul-envs.sh
```

### 1. Create a new container's config

The idea is to create a separate container for each project (that will rely on a
same base container with variations).

This container first need to be configured to point to your project and have the
right arguments (e.g. the right tools and git configuration). This is done
through the `paul-envs.sh create` "command".

First ensure the target project is present locally in your host, then run:
```sh
./paul-envs.sh create <path/to/your/project>
```

Optionally, you may add a lot of flags to better configure that container.
Here's an example of a real-life usage:
```sh
# Will create a container named `myapp` with a default `zsh` shell and many
# configurations. Also mount your `.git-credentials` readonly to the container.
./paul-envs.sh create myapp ~/work/api \
  --name myProject
  --shell zsh \
  --node-version 22.11.0 \
  --git-name "John Doe" \
  --git-email "john@example.com" \
  --port 8000 \
  --port 5432 \
  --packages "fzf ripgrep" \
  --volume ~/.git-credentials:/home/dev/.git-credentials:ro
```

Without the corresponding flags, prompts will be proposed by `paul-envs.sh` for
important parameters (choosen shell, wanted pre-mounted volumes etc.).

What this step does is just to create both a `yaml` and a `.env` file containing
your container's configuration. It doesn't build anything yet.

Those files will be written in a new `./projects/<NAME>` directory, with
`<NAME>` being the name you chose under the `--name` flag or the directory's
name by default. Those files can be directly edited if you want.

### 2. Build the container

The previous file created both a "compose" and "env" file - basically
configuration to define the container we want to build.

This step relies on `docker compose`, which you should have locally installed.

To build a container, just run the `paul-envs.sh build <NAME>` command.
For example, with a container named `myApp`, you would just do:
```sh
./paul-envs.sh build myApp
```

This will take some time as the initialization of the container is going on:
packages are loaded, tools are set-up etc.

### 3. Run the container

Now that the container is built. It can be run at any time, with the
`./paul-envs.sh run` command.
For example, with a container named `myApp`, you would do:
```sh
./paul-envs.sh build myApp
```

You will directly switch to that container's `$HOME/projects/<NAME>` directory
(e.g. for a `myApp` project with a `dev` default username it will be
`/home/dev/projects/myApp`) which contains the project you linked to that
container.

You can go out of that container at any time (e.g. by calling `exit`), as you
exit that container, everything that is not part of the "persisted volume" (see
`What gets preserved vs. ephemeral` chapter) is reset to the state it was at
build-time.

As cache and tools' data directory are still persisted, you might also want to
re-build the container if you feel that you need a completely fresh state. This
should be needed extremely rarely hopefully (you more probably will want to
rebuild just to update some base tools).

### Other commands

`paul-envs.sh` also proposes the `list` and `remove` commands, respectively to
list "created" configurations (what's in the `projects` directory basically) and
to easily remove one of them (basically a `rm` command for that configuration)
respectively:
```sh
# List all created configurations, built or not
./paul-envs.sh list

# Remove the configuration file for the `myApp` container
./paul-envs.sh remove myApp
```

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

## Deep dive on how it works

Much like container applications, this repository is organized in separate
layers: `Dockerfile`, `compose.yaml` and `paul-envs.sh` script, from the core
layer to the most outer one, each inner layer being able to run independently
of its outer layers (just losing some features in the process).

The following chapters explain each layer and how to run them independently if
wanted. If you just want to [run this without understanding every little
details](https://www.youtube.com/watch?v=bJHPfpOnDzg) just run `paul-envs.sh`
and performs the operations it advertises.

### Dockerfile

The `Dockerfile` sets a simple Ubuntu LTS environment with a shell of your
preference (either `bash` as default or `zsh` or `fish`) and optional popular
CLI tools (`neovim`, `starship`, `atuin`, `zellij`, `jujutsu` and `mise`).

It also copies the content of the `configs` directory inside of that container
and sets-up `$HOME/.container-cache` and `$HOME/.container-local` directories
for cache and tools' local data (including shell history, `atuin` database,
`mise` environments) respectively.

You can rely on this `Dockerfile` without anything else, as a standalone, e.g.
via `docker`. Note that if you do that, you won't have persistent volumes for
cache, tools data and the project code, which would have to be re-populated each
time the container is run. Adding persistence is the main point of the
`compose.yaml` file.

### The configs directory

The `configs` directory in this repository helps with the initialization of
so-called "dotfiles" in the created containers.

Its content will be merged with the home directory of the container. As such you
can put a `.bashrc` directly in there at its root, and the config for the tools
you planned to install (e.g. the `starship` configuration file: `starship.toml`,
a `nvim` directory for `neovim` etc.):
```
configs/
├── .bashrc
└── .config/
    ├── starship.toml (config for the starship tool)
    └── nvim/
        └── ... (your neovim config)
```

All files in `configs` will be copied as is unmodified, with two exceptions:

1.  shell files  (`.bashrc`, `.zshrc` and/or `.config/fish/config.fish` files)
    may still be updated after being copied in the dockerfile to redirect their
    history to `~/.container-local` (see next chapter) to ensure history
    persistence.

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

The job of copying the `configs` directory's content is taken by the
`Dockerfile`. Meaning that you'll profit from this even if you're not relying on
`docker compose` or `paul-envs.sh`.

#### Note about neovim

If you set a `neovim` config with plugins, they will be pre-installed as the
container is built if you rely on the `lazy.nvim` plugin manager.

With other solutions, the installation will need to be done the first time the
container is ran (it should be persisted thereafter if going the
`compose.yaml` or `paul-envs.sh` route).

### compose.yaml

The `compose.yaml` file allows `docker compose` to build a container with
the right arguments for your project. More importantly, it also mount the
right "volumes" so that some changes (project changes, cache, tools data,
shell history etc.) are persisted.

In simple single-projects scenarios, it can also be relied on directly.
Just set the right env variables listed in there (.e.g in an `env` file) and
rely on `docker compose` directly (e.g. `docker compose build`). It works!

If you just want to do that, refer to the `compose.yaml` file. It contains
documentation on how to exploit `docker compose` directly instead of going
through my `paul-envs.sh` script.

### The paul-envs.sh script

Managing very dynamic configurations for multiple projects just with
`docker compose` is not as straightforward as I would have liked: depending on
what you want to do, the idiomatic ways to configure it are through either
environment variables or new compose files.

Instead of doing both, which would have been difficult to maintain (and to
remember what goes where and why), I thus decided to create a `paul-envs.sh`
script whose job is to wrap both compose files creation and `docker compose`
calls.

Through a small list of commands and a high number of flags, it is now possible
to easily create configurations, build containers, run them, list them etc.

That script actually just writes `compose.yaml` files and wraps `docker compose`
calls, with also some input validation and the printing of helpful information
on top.

### persisted volumes

Two container "volumes", a `~/.container-cache` and a `~/.container-local`
directory, will be present in the container.
Their main specificity is that unlike almost anything else, their contents are
persisted through multiple containers run (unless you went directly the
`Dockerfile` route).

The former (`.container-cache`) is configured to store the various "caches"
(e.g. `npm` and `yarn` loaded package cache, or any other similar cache), to
prevent re-doing the same avoidable requests/operations each time a container is
spawned.

The latter (`.container-local`) is intended for persistent tool data instead,
such as shell history, neovim undo history, tool databases etc.

The dockerfile is configured so that the installed tools know they have to use
those directories for the aforementioned purposes.
_I made use both of the XDG spec and of tool-specific configuration for this._

Along the mounted project, those are the only directories which are persisted.

## TODO:

- Add Kakoune and Helix as potential tools
