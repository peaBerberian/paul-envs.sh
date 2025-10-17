# Containers

## Quick Start

1. Copy `.env.example` to `.env` and configure it
2. Populate the `configs/` directory with your dotfiles (merged with `$HOME`)
3. Build: `docker compose build`
4. Run: `docker compose run --rm devenv`

## What's this

I often have to switch between projects at work.

Some of those projects are very fast-moving JS projects with a lot of
dependencies churning, moving at a pace where I cannot guarantee my
trust in everything that's going on (that their updates or dependencies work
well on my system, that they have a limited impact on it and not break anything
else etc.).

Most of those developers also have very similar environments between each
other, which is sadly not close to mine, so I encounter a lot of issues
(mostly linked to their bash scripts and such) that they never encounter.

I thus decided to rely on a simple minimal container, adding my current
developing tools to it, and do development and scripting on it when working on
their project.
As my setup is only terminal-based (neovim, CLI tools), this can be done
relatively easily.

At first I was just relying on `systemd-nspawn`, as this is a tool that I knew.
But by using it for this, I thought that having a base with an ephemeral
"overlay" and a mounted "volume" from the host for the source code was the most
flexible solution for my setup, so I ended up with a more complex `Dockerfile`
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

### cache directory

The `cache` directory found here will be used by the container as a persistent
cache (e.g. `npm` / `yarn` cache)/history (e.g. shell history, `nvim` history,
`zoxide`/`atuin` databases if enabled) for tools running inside it.
It shouldn't be updated on the host but can be emptied to ensure that you
start fresh - you probably want to keep some kind of peristent cache to speed
up recurrent tasks though.

Tools installed on the container are already configured to create entries in
this directory.

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
be removed when the container is exited) or "persistent" (host files as read-only
and kept as-is).

- **Preserved**: the mounted project directory (`~/projects/app`) and the
  "cache" directory (mounted as `~/.container-cache`).

- **Persistent**: Git credentials if `GIT_CREDS_HOST` is set in `.env`

- **Ephemeral**: All other changes (further installed packages, tool
  configuration etc.)

If a new element needs to be added to the container outside the mounted project
directory, the dockerfile will need to be updated and re-built.
