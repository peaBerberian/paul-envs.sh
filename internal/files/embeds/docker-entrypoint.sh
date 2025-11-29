#!/bin/bash

# This is the container's entry point. All containers will start executiing
# that script to ensure that everything is initialized and starting the right
# daemons if needed.
# Note that is is executed as root, as it needs enough permissions to e.g. start
# an ssh daemon if wanted.
#
# It then executes either the default shell (if executed without arguments) or
# the arguments given to it.

DOCKER_USERNAME="${DOCKER_USERNAME:-dev}"
USER_SHELL="${USER_SHELL:-/usr/bin/bash}"
INITIAL_CACHE_DIR=${INITIAL_CACHE_DIR:-/home/${DOCKER_USERNAME}/.initial-cache}
INITIAL_LOCAL_DIR=${INITIAL_LOCAL_DIR:-/home/${DOCKER_USERNAME}/.initial-local}
DOCKER_CACHE_DIR=${DOCKER_CACHE_DIR:-/home/${DOCKER_USERNAME}/.container-cache}
DOCKER_LOCAL_DIR=${DOCKER_LOCAL_DIR:-/home/${DOCKER_USERNAME}/.container-local}
CACHE_MARKER="${DOCKER_CACHE_DIR}/.initialized"
LOCAL_MARKER="${DOCKER_LOCAL_DIR}/.initialized"

# Initialize shared cache (only if not already initialized by another container)
if [ ! -f "$CACHE_MARKER" ]; then
    echo "Initializing shared cache..."
    mkdir -p "$DOCKER_CACHE_DIR"
    cp -a "$INITIAL_CACHE_DIR/." "$DOCKER_CACHE_DIR/" 2>/dev/null || true
    touch "$CACHE_MARKER"
fi

# Initialize local state (per-project, always check)
if [ ! -f "$LOCAL_MARKER" ]; then
    echo "Initializing local state..."
    mkdir -p "$DOCKER_LOCAL_DIR"
    cp -a "$INITIAL_LOCAL_DIR/." "$DOCKER_LOCAL_DIR/" 2>/dev/null || true
    touch "$LOCAL_MARKER"
fi

# SSH daemon setup
if [[ -d /var/run/sshd ]] && ! pgrep -x sshd >/dev/null; then
    /usr/sbin/sshd -D &
    if [[ -t 0 ]] && [[ $# -eq 0 ]]; then
        IP=$(hostname -I | awk "{print \$1}")
        echo "NOTE: Listening for ssh connections at ${DOCKER_USERNAME}@${IP}:22"
    fi
fi

# Execute command or start shell
if [[ $# -eq 0 ]]; then
    exec su ${DOCKER_USERNAME} -s ${USER_SHELL}
else
    exec runuser -u ${DOCKER_USERNAME} -- "$@"
fi
