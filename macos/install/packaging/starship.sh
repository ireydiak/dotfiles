#!/bin/bash

curl -sS https://starship.rs/install.sh | sh
cp $CONFIG_DIR/starship/starship.toml ~/.config/starship.toml
