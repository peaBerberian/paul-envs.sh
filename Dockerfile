FROM ubuntu:24.04 AS ubuntu-base

# Configurable user settings
ARG HOST_UID=1000
ARG HOST_GID=1000
ARG USERNAME=dev
ARG USER_SHELL=bash

# Install base packages
RUN apt-get update && apt-get install -y \
  git \
  curl \
  build-essential \
  bash \
  unzip \
  && rm -rf /var/lib/apt/lists/*

# Install optional shells
RUN if [ "$USER_SHELL" = "fish" ]; then \
  apt-get update && apt-get install -y fish && rm -rf /var/lib/apt/lists/*; \
  elif [ "$USER_SHELL" = "zsh" ]; then \
    apt-get update && apt-get install -y zsh && rm -rf /var/lib/apt/lists/*; \
    echo -e "\nexport HISTFILE=/home/${USERNAME}/.container-local/.zsh_history" >> /home/${USERNAME}/.zshrc; \
  elif [ "$USER_SHELL" = "bash" ]; then \
    echo -e "\nexport HISTFILE=/home/${USERNAME}/.container-local/.bash_history" >> /home/${USERNAME}/.bashrc; \
  fi

# Create user
RUN if id -u ubuntu >/dev/null 2>&1; then userdel -r ubuntu; fi && \
  groupadd -g ${HOST_GID} ${USERNAME} && \
  useradd -u ${HOST_UID} -g ${HOST_GID} -m -s /usr/bin/${USER_SHELL} ${USERNAME}

USER ${USERNAME}

ENV SHELL=/usr/bin/${USER_SHELL}

# Set various persistent caches locations through env
ENV XDG_CACHE_HOME=/home/${USERNAME}/.container-cache/cache \
    XDG_STATE_HOME=/home/${USERNAME}/.container-local/state \
    XDG_DATA_HOME=/home/${USERNAME}/.container-local/data

#############################################
FROM ubuntu-base AS ubuntu-tools

# Additional packages outside the core base, separated by a space.
# Have to be in Ubuntu's default repository
ARG SUPPLEMENTARY_PACKAGES=""

# Configurable tool installation
ARG INSTALL_NEOVIM=true
ARG INSTALL_STARSHIP=true
ARG INSTALL_ATUIN=true
ARG INSTALL_MISE=true
ARG NODE_VERSION=22.11.0
ARG GIT_AUTHOR_NAME
ARG GIT_AUTHOR_EMAIL

USER root

ENV _ZO_DATA_DIR=/home/${USERNAME}/.container-local/zoxide \
    STARSHIP_CACHE=/home/${USERNAME}/.container-local/starship \
    ATUIN_DB_PATH=/home/${USERNAME}/.container-local/atuin/history.db

RUN apt-get update && apt-get install -y \
  $SUPPLEMENTARY_PACKAGES \
  && rm -rf /var/lib/apt/lists/*

# Install Neovim (optional)
RUN if [ "$INSTALL_NEOVIM" = "true" ]; then \
    curl -LO https://github.com/neovim/neovim/releases/latest/download/nvim-linux-x86_64.tar.gz && \
    tar -C /opt -xzf nvim-linux-x86_64.tar.gz && \
    rm nvim-linux-x86_64.tar.gz && \
    ln -s /opt/nvim-linux-x86_64/bin/nvim /usr/local/bin/nvim; \
  fi

# Install Starship (optional)
RUN if [ "$INSTALL_STARSHIP" = "true" ]; then \
    curl -sS https://starship.rs/install.sh | sh -s -- -y; \
  fi

USER ${USERNAME}

RUN git config --global credential.helper store
RUN git config --global merge.conflictstyle zdiff3
RUN git config --global user.name "$GIT_AUTHOR_NAME"
RUN git config --global user.email "$GIT_AUTHOR_EMAIL"

# Install Atuin (optional)
RUN echo "INSTALL_ATUIN value: '$INSTALL_ATUIN'" && \
  if [ "$INSTALL_ATUIN" = "true" ]; then \
    curl --proto '=https' --tlsv1.2 -sSf https://setup.atuin.sh | bash; \
  fi

# Copy config files
RUN --mount=type=bind,source=configs,target=/tmp/configs \
  cp -r /tmp/configs/. /home/${USERNAME}/ && \
  chown -R ${USERNAME}:${USERNAME} /home/${USERNAME}

# Install mise (optional)
# `nvm` is mainly maintained by ljharb, enough said. Also, it doesn't
# even work with the fish shell which happens to be my shell.
# I use mise instead here, but I'm sure good other solutions exist.
RUN if [ "$INSTALL_MISE" = "true" ]; then \
    # 1. Install mise (https://mise.jdx.dev/installing-mise.html)
    curl https://mise.jdx.dev/install.sh | sh && \
    # 2. Activate mise for the chosen shell
    if [ "$USER_SHELL" = "fish" ]; then \
      echo 'mise activate fish | source' >> /home/${USERNAME}/.config/fish/config.fish; \
    elif [ "$USER_SHELL" = "bash" ]; then \
      echo 'eval "$(mise activate bash)"' >> /home/${USERNAME}/.bashrc; \
    elif [ "$USER_SHELL" = "zsh" ]; then \
      echo 'eval "$(mise activate zsh)"' >> /home/${USERNAME}/.zshrc; \
    fi && \
    # 3. Install Node, set global default, install yarn (non-interactive)
    bash -c 'export PATH="$HOME/.local/bin:$PATH" && \
             mise use -g node@${NODE_VERSION} && \
             mise exec -- npm config set prefix "$HOME/.local" && \
             mise exec -- npm config set cache /home/${USERNAME}/.container-cache/.npm && \
             mise exec -- npm install -g yarn && \
             mise exec -- yarn config set cacheFolder /home/${USERNAME}/.container-cache/.yarn'; \
  fi

USER root

RUN if [ "$INSTALL_MISE" != "true" ]; then \
    # Just install nodejs and npm from Ubuntu's repositories
    apt-get update && apt-get install -y \
      nodejs \
      npm \
      && npm config set prefix "$HOME/.local" \
      && npm config set cache /home/${USERNAME}/.container-cache/.npm \
      && npm install -g yarn \
      && yarn config set cacheFolder /home/${USERNAME}/.container-cache/.yarn \
      && rm -rf /var/lib/apt/lists/*; \
  fi

USER ${USERNAME}

# Pre-install nvim plugins if neovim is installed and config exists
RUN if [ "$INSTALL_NEOVIM" = "true" ] && [ -d /home/${USERNAME}/.config/nvim ]; then \
      nvim --headless "+Lazy! sync" +qa || true; \
  fi

#############################################
FROM ubuntu-tools AS ubuntu-projects

USER ${USERNAME}

# Set-up projects directory
RUN mkdir -p /home/${USERNAME}/projects

WORKDIR /home/${USERNAME}/projects

CMD $SHELL
