#!/bin/bash

brew install koekeishiya/formulae/yabai

cp -r $CONFIG_DIR/yabai ~/.config/yabai
cp $CONFIG_DIR/yabai/yabairc ~/.yabairc

yabai --start-service
