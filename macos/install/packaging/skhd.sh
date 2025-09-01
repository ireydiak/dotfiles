#!/bin/bash

brew install jackielii/tap/skhd-zig

rm -rf ~/.config/shkd

cp -r $CONFIG_DIR/skhd ~/.config/shkd

skhd --install-service
skhd --start-service
