#!/bin/bash

set -eE

BASE_DIR="$HOME/.local/share/ireydiak/macos"
INSTALL_DIR="$BASE_DIR/install"
CONFIG_DIR="$BASE_DIR/config"

# Preparation
source $INSTALL_DIR/preflight/set-env.sh
source $INSTALL_DIR/preflight/fonts.sh
source $INSTALL_DIR/preflight/zsh.sh
source $HOME/.zshrc
source $INSTALL_DIR/preflight/package-manager.sh
source $HOME/.zshrc

# Install packages and configure them
# Must be executed first
#
for dir in "$INSTALL_DIR"/packaging/*.sh; do
    source "$INSTALL_DIR/$dir"
done

echo "Installation completed! Restart this shell to access installed apps"
