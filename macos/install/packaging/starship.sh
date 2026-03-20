#!/bin/bash

if [[ ! -d "/usr/local/bin" ]]; then
    sudo mkdir -p /usr/local/bin
    sudo chown -R $(whoami):staff /usr/local/bin
fi

curl -sS https://starship.rs/install.sh | sh
cp $CONFIG_DIR/starship/starship.toml ~/.config/starship.toml
