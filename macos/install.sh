#!/bin/bash

set -eE

BASE_DIR="$HOME/.local/share/ireydiak/macos"
INSTALL_DIR="$BASE_DIR/install"
CONFIG_DIR="$BASE_DIR/config"

# Preparation
source $INSTALL_DIR/preflight/set-env.sh
source $INSTALL_DIR/preflight/fonts.sh
source $INSTALL_DIR/preflight/package-manager.sh

# Install packages and configure them
# Must be executed first
source $INSTALL_DIR/packaging/packages.sh

source $INSTALL_DIR/packaging/ghostty.sh
source $INSTALL_DIR/packaging/lazyvim.sh
source $INSTALL_DIR/packaging/nvm.sh
source $INSTALL_DIR/packaging/omarchy.sh
source $INSTALL_DIR/packaging/sketchybar.sh
source $INSTALL_DIR/packaging/skhd.sh
source $INSTALL_DIR/packaging/starship.sh
source $INSTALL_DIR/packaging/yabai.sh

echo "Installation completed! Restart this shell to access installed apps"
