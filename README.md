# Containers

## Quick Start

1. Copy `.env.example` to `.env` and configure it
2. Populate the `configs/` directory with your dotfiles (merged with `$HOME`)
3. Build: `docker compose build`
4. Run: `docker compose run --rm devenv`

## What's this

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

## How it works

The `Dockerfile` and `compose.yaml` are updated in function of tools I'm using.
What they do is just set a minimal Ubuntu environment with those.

Note that including many other things, it relies on neovim with the Lazy plugin
manager (note to self as I may not keep using that one long-term).

### .env file

An `.env` file should be created (based e.g. on `.env.example`) to setup
environment variables, allowing to setup the default shell, which tool are
enabled, the user name, the preferred node.js version, the basic git config
(name + e-mail address) and even the projects to mount inside the container
(path on the host to the source code to work on).

### configs directory

A `configs` directory is also present and can be updated on the host. It should
contain the config files for tools that I currently set-up in the `Dockerfile`.

Its content will be merged with the home directory of the container. As such you
can put a `.bashrc` directly in there at its root, or directories in it such as
`.config/nvim` for a neovim config:
```
configs/
├── .bashrc
└── .config/
    └── nvim/
        └── ... (your neovim config)
```

### persisted volumes

Two container "volumes", a `~/.container-cache` and a `~/.container-local`
directory, will be present in the container.
Their main specificity is that unlike almost anything else, their contents are
persisted through multiple containers run.

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
If an instability arises at some point, the "volumes" corresponding to those
directories cache can be reset. For example with docker-compose:
```sh
# Nuclear option: clear all caches
docker compose down -v  # stops and removes volumes
docker compose build    # rebuild container
docker compose run --rm devenv
```

## How to run it

Once setup is done (configs and env files), the container needs first to be
"built".

With docker-compose, this can be done just by `cd`-ing in this directory, then
calling:
```sh
docker compose build
```

This can be re-done later to refresh the installed build tools, if needed
(hopefully not a lot).

Then anytime there's need to develop on the project, that container can be
"run" by just running:
```sh
docker compose run --rm devenv
```

With `--rm` meaning that the container will be removed on exit. What this imply
is that all modifications done in the base container which are not part of the
mounted project's directory will disappear when that container is not
relied on anymore - this is actually one of the point of this setup to always
start fresh from a known stable config.

## What gets preserved vs. ephemeral

When working inside the container, here's what you can expect to be either
"preserved" (changes will stay from container to container), "ephemeral" (it will
be removed when the container is exited) or "persistent" (host files mounted as
read-only and kept as-is).

- **Preserved**: the mounted project directory (`~/projects/app`), the
  "cache" directory (mounted as `~/.container-cache`) and the "local" directory
  (mounted as `~/.container-cache`) - see the "persisted volumes" chapter for
  those last two.

- **Persistent**: Git credentials if `GIT_CREDS_HOST` is set in `.env`

- **Ephemeral**: All other changes (further installed global packages, global
  system configurations etc.)

If a new element needs to be added to the container outside the mounted project
directory, the dockerfile will need to be updated and re-built.
