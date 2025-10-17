FROM ubuntu:24.04

# Configurable user settings
ARG HOST_UID=1000
ARG HOST_GID=1000
ARG USERNAME=dev
ARG USER_SHELL=bash

# Configurable tool installation
ARG INSTALL_NEOVIM=true
ARG INSTALL_STARSHIP=true
ARG INSTALL_ATUIN=true
ARG INSTALL_MISE=true
ARG INSTALL_EZA=true
ARG INSTALL_ZOXIDE=true
ARG NODE_VERSION=22.11.0

# Install base packages
RUN apt-get update && apt-get install -y \
    sudo \
    git \
    ripgrep \
    curl \
    unzip \
    build-essential \
    bash \
    nodejs \
    fzf \
    npm \
    && rm -rf /var/lib/apt/lists/*

# Install optional shells
RUN if [ "$USER_SHELL" = "fish" ]; then \
        apt-get update && apt-get install -y fish && rm -rf /var/lib/apt/lists/*; \
    elif [ "$USER_SHELL" = "zsh" ]; then \
        apt-get update && apt-get install -y zsh && rm -rf /var/lib/apt/lists/*; \
    fi

# Create user
RUN if id -u ubuntu >/dev/null 2>&1; then userdel -r ubuntu; fi && \
    groupadd -g ${HOST_GID} ${USERNAME} && \
    useradd -u ${HOST_UID} -g ${HOST_GID} -m -s /usr/bin/${USER_SHELL} ${USERNAME}

RUN --mount=type=bind,source=configs,target=/tmp/configs \
    mkdir -p /home/${USERNAME}/.config && \
    if [ -d /tmp/configs/fish ]; then \
        cp -r /tmp/configs/fish /home/${USERNAME}/.config/fish; \
    fi && \
    if [ -d /tmp/configs/nvim ]; then \
        cp -r /tmp/configs/nvim /home/${USERNAME}/.config/nvim; \
    fi && \
    if [ -f /tmp/configs/.bashrc ]; then \
        cp -r /tmp/configs/.bashrc /home/${USERNAME}/.bashrc; \
    fi && \
    if [ -f /tmp/configs/.zshrc ]; then \
        cp -r /tmp/configs/.zshrc /home/${USERNAME}/.zshrc; \
    fi && \
    if [ -f /tmp/configs/starship.toml ]; then \
        cp -r /tmp/configs/starship.toml /home/${USERNAME}/.config/starship.toml; \
    fi && \
    chown -R ${USERNAME}:${USERNAME} /home/${USERNAME}/.config

# Install eza (optional)
# TODO SUPPLEMENTARY_PACKAGES instead?
RUN if [ "$INSTALL_EZA" = "true" ]; then \
  apt-get update && \
    (command -v eza >/dev/null 2>&1 || apt-get install -y eza || true) && \
    rm -rf /var/lib/apt/lists/*; \
  fi

# Install zoxide (optional)
RUN if [ "$INSTALL_ZOXIDE" = "true" ]; then \
  apt-get update && \
    (command -v zoxide >/dev/null 2>&1 || apt-get install -y zoxide || true) && \
    rm -rf /var/lib/apt/lists/*; \
  fi

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

ENV SHELL=/usr/bin/${USER_SHELL}

RUN git config --global credential.helper store
RUN git config --global merge.conflictstyle zdiff3

RUN git config --global user.name "$GIT_AUTHOR_NAME"
RUN git config --global user.email "$GIT_AUTHOR_EMAIL"

# Set-up directories
RUN mkdir -p /home/${USERNAME}/projects

WORKDIR /home/${USERNAME}/projects

# Install Atuin (optional)
RUN echo "INSTALL_ATUIN value: '$INSTALL_ATUIN'" && \
  if [ "$INSTALL_ATUIN" = "true" ]; then \
    curl --proto '=https' --tlsv1.2 -sSf https://setup.atuin.sh | bash; \
  fi

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
             npm config set prefix "$HOME/.local" && \
             npm install -g yarn'; \
  fi


# Pre-install nvim plugins if neovim is installed and config exists
RUN if [ "$INSTALL_NEOVIM" = "true" ] && [ -d /home/${USERNAME}/.config/nvim ]; then \
        nvim --headless "+Lazy! sync" +qa || true; \
    fi

CMD $SHELL
